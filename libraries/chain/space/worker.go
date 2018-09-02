/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2018.4.28
 * Website    : http:www.gocoin.com
 * Function   : transaction build block to network
**/
package space

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/chain/space/state"
	"gocoin/libraries/common"
	"gocoin/libraries/crypto/sha3"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/event"
	"gocoin/libraries/rlp"
	//"gocoin/libraries/log"

	"gocoin/libraries/chain/blsgroup"
	"gocoin/libraries/chain/transaction/gwallet"
	"gocoin/libraries/chain/transaction/types"
	"gocoin/libraries/chain/validategroup"
	"gocoin/libraries/chain/vrf"
)

var (
	LiquidationEmptyKey = []byte("LData")
	EcdEmptyKey         = []byte("Ecd")
	PublishEmptyKey     = []byte("Publish")
)

func rlpSigHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type BlockSigInfo struct {
	BlockHash   common.Hash
	Time        *big.Int
	Sender      common.Address
	Percent     []byte
	PreknownSig *blsgroup.BlsSign
}

type Worker struct {
	chainDb       lvldb.Database
	txpool        *TxPool
	spaceChain    *BlockChain
	bls           *blsgroup.BlsGroup
	validateGroup *validategroup.ValidateGroup

	lastBlockFeed event.Feed
	blockSigFeed  event.Feed
	newBlockFeed  event.Feed
	scope         event.SubscriptionScope

	SigInfoCh   chan BlockSigEvent
	lastBlockCh chan LastBlockEvent
	blockHashs  map[common.Hash]struct{}
	stop        chan struct{}

	txs       []*Block_Tx
	sigBlocks map[common.Hash]*protocol.Block
	sigInfos  map[common.Address]*BlockSigInfo
	currBlock *protocol.Block

	gw        *gwallet.GWallet
	blockVrf  *vrf.Vrf
	threshold uint64

	sender             common.Address
	fixedAddress       common.Address
	coefficient        float32
	sortIndex          int
	coefficientMax     float32
	receiveBlockIsTrue bool
	blockIsTrue        bool
	fixed              bool

	mu      sync.Mutex
	blockmu sync.Mutex
	txsmu   sync.Mutex
	atWork  int32
}

func NewWorker(chain *BlockChain, pool *TxPool, validate *validategroup.ValidateGroup, db lvldb.Database) *Worker {
	worker := &Worker{
		coefficient:        float32(0.0),
		sortIndex:          int(0),
		coefficientMax:     float32(0.0),
		threshold:          uint64(1000),
		receiveBlockIsTrue: false,
		blockIsTrue:        false,
		fixed:              false,
		chainDb:            db,
		txpool:             pool,
		spaceChain:         chain,
		validateGroup:      validate,
		SigInfoCh:          make(chan BlockSigEvent, 32),
		lastBlockCh:        make(chan LastBlockEvent, 4),
		sigBlocks:          make(map[common.Hash]*protocol.Block),
		sigInfos:           make(map[common.Address]*BlockSigInfo),
		blockHashs:         make(map[common.Hash]struct{}),
		stop:               make(chan struct{}),
	}

	return worker
}

func (self *Worker) Start() {

	self.mu.Lock()
	defer self.mu.Unlock()

	if atomic.LoadInt32(&self.atWork) == 1 {
		return
	}

	atomic.StoreInt32(&self.atWork, 1)

	go self.update()
	go self.loop()
}

func (self *Worker) Stop() {

	self.mu.Lock()
	defer self.mu.Unlock()
	if atomic.LoadInt32(&self.atWork) == 1 {
		self.stop <- struct{}{}
	}

	atomic.StoreInt32(&self.atWork, 0)
}

func (self *Worker) SetGroupSize(k uint64) {
	self.threshold = k
}

func (self *Worker) ResetValidateGroup(number uint64) {

	addrs := self.validateGroup.SelectConsensus(number)
	self.fixed = false
	fmt.Println("ValidateGroup Size : ", len(addrs))

	if len(addrs) > 0 {
		if len(addrs) > 0 && len(addrs) < 3 {
			self.fixed = true
			self.fixedAddress = addrs[0]
			fmt.Println("Space Worker fixed block address :  ", self.fixedAddress)
		}
	}

}

func (self *Worker) SetSenderAddress(address common.Address) {

	fmt.Println(" Sender : ", address)
	self.sender = address
}

func (self *Worker) AddSigBlock(siginfo BlockSigEvent) {
	fmt.Println("revice Sig :", len(self.sigInfos))
	self.SigInfoCh <- siginfo
}

func (self *Worker) SetLastBlockCh(block LastBlockEvent) {

	fmt.Println("revice LastBlock :", block.Block.GetNumberU64(), " Hash : ", block.Hash)
	_, ok := self.blockHashs[block.Hash]
	if ok {
		return
	} else {
		self.blockHashs[block.Hash] = struct{}{}
	}
	self.lastBlockCh <- block
}

func (self *Worker) update() {

	for {
		select {
		case lastblock := <-self.lastBlockCh:
			{
				self.bls = blsgroup.GetGroup(uint(0))
				self.gw = gwallet.GetGWallet(0)
				self.threshold = uint64(self.bls.Threshold())
				if lastblock.Hash == lastblock.Block.Hash() {
					head := lastblock.Block.BlockHead()
					hash := head.HashNoSig()

					addre, err := self.gw.SignSender(hash, head.Sign)
					if err != nil {
						fmt.Println("Sig error : ", err)
						return
					}

					block := self.spaceChain.CurrentBlock()
					if block != nil {

						addrs := self.validateGroup.SelectConsensus(lastblock.Block.GetNumberU64())
						var sendsign bool = false
						if len(addrs) > 1 {

							for _, addr := range addrs {
								if addr == self.sender {
									sendsign = true
								}
							}
						}

						if sendsign {

							self.blockVrf = vrf.NewVrf(0, self.sender, addrs)
							self.bls = blsgroup.GetGroup(uint(0))
							coefficient, _ := self.blockVrf.ValidateBlock(block.GetNumberU64(), block.Hash(), addre)
							//fmt.Println("Percent : ", coefficient, "head.Percent : ", types.ByteToFloat32(head.Percent))
							if math.Abs(float64(types.ByteToFloat32(head.Percent)-coefficient)) < float64(0.00001) {

								//fmt.Println("Validate Block")
								if self.validateBlock(lastblock.Block, false) {
									self.receiveBlockIsTrue = true
									if self.coefficientMax < types.ByteToFloat32(head.Percent) {
										self.coefficientMax = types.ByteToFloat32(head.Percent)
										self.sigBlocks[hash] = lastblock.Block

										siginfo := &BlockSigInfo{
											BlockHash: hash,
											Time:      big.NewInt(time.Now().Unix()),
											Percent:   head.Percent,
											Sender:    self.sender,
										}

										siginfo.PreknownSig = self.bls.Sign(siginfo.BlockHash.Bytes())
										self.sigInfos[self.sender] = siginfo
										ev := BlockSigEvent{rlpSigHash(siginfo), siginfo}
										self.blockSigFeed.Send(ev)
										self.SigInfoCh <- ev
									}
								}
							}
						}

					}
				}
			}
		case siginfo := <-self.SigInfoCh:
			{
				self.bls = blsgroup.GetGroup(uint(0))
				self.threshold = uint64(self.bls.Threshold())
				currBlock, ok := self.sigBlocks[siginfo.Sig.BlockHash]
				if ok {
					addrs := self.validateGroup.SelectConsensus(currBlock.GetNumberU64())
					var sendsign bool = false
					if len(addrs) > 1 {

						for _, addr := range addrs {
							if addr == self.sender {
								sendsign = true
							}
						}
					}

					if sendsign {
						sig, ok := self.sigInfos[siginfo.Sig.Sender]
						if ok {
							if types.ByteToFloat32(sig.Percent) < types.ByteToFloat32(siginfo.Sig.Percent) {
								self.sigInfos[siginfo.Sig.Sender] = siginfo.Sig
							}
						} else {
							self.sigInfos[siginfo.Sig.Sender] = siginfo.Sig
						}

						if len(self.sigInfos) >= int(self.threshold) {

							var (
								signs  []blsgroup.BlsSign
								sigmap map[common.Hash]uint64
							)
							sigmap = make(map[common.Hash]uint64)
							for _, vsb := range self.sigInfos {
								count, ok := sigmap[vsb.BlockHash]
								if ok {
									sigmap[vsb.BlockHash] = count + uint64(1)
								} else {
									sigmap[vsb.BlockHash] = uint64(1)
								}
							}

							for hash, size := range sigmap {
								if size >= self.threshold {

									self.blockmu.Lock()
									for _, vsb := range self.sigInfos {
										if hash == vsb.BlockHash {
											signs = append(signs, *vsb.PreknownSig)
										}
									}
									for sender, _ := range self.sigInfos {
										delete(self.sigInfos, sender)
									}
									suc, sig := self.bls.Verigy(signs, hash.Bytes())
									if suc {

										self.coefficientMax = 0.0
										currBlock, ok := self.sigBlocks[hash]
										if ok {
											currBlock.SetSigner(sig)
											//fmt.Println("block bls sig msg  :", hash.Bytes(), "block bls sign :", sig)
											fmt.Println("bls suc  Space Block : ", currBlock.GetNumberU64(), "blockHash : ", currBlock.Hash())
											ev := NewBlockEvent{currBlock.Hash(), currBlock}
											self.newBlockFeed.Send(ev)

											go self.commitNewBlock(currBlock)
										}
										self.sigBlocks = make(map[common.Hash]*protocol.Block)
									}
									self.blockmu.Unlock()
								}
							}
						}
					}
				}

			}
		case <-self.stop:
			return
		}
	}
}

func (self *Worker) timeoutSubmission() {

	forceSync := time.NewTicker(2 * time.Second)
	defer forceSync.Stop()

	var (
		bNumber int = 0
	)

	for {
		select {
		case <-forceSync.C:
			{
				bNumber++
				if self.blockIsTrue {
					if !self.receiveBlockIsTrue {
						if self.sortIndex >= bNumber-2 {
							self.blockIsTrue = false
							ev := LastBlockEvent{self.currBlock.Hash(), self.currBlock}
							self.lastBlockFeed.Send(ev)
							break
						}

					} else {
						break
					}
				}
			}
		}
	}
}

func (self *Worker) loop() {

	for {
		block := self.spaceChain.CurrentBlock()
		if block != nil {

			if block.GetNumberU64() == 0 {
				wait := time.Duration(int64(28)) * time.Second
				time.Sleep(wait)
			} else {

				now := time.Now().Unix()
				tstamp := block.Timestamp().Int64() + 28
				if tstamp-now > 0 {
					wait := time.Duration(tstamp-now) * time.Second
					time.Sleep(wait)
				}
			}

			if self.txpool.GetTxsCount() > 0 {
				addrs := self.validateGroup.SelectConsensus(block.GetNumberU64() + 1)
				var sendsign bool = false
				fmt.Println("Space  validateGroup  size : ", len(addrs), "validateGroup Address :", addrs, " Sender : ", self.sender)
				if len(addrs) > 0 {

					for _, addr := range addrs {
						if addr == self.sender {
							sendsign = true
						}
					}

					if sendsign {
						fmt.Println("**************   buildBlock  *************** tx size : ", len(self.txs), "Time : ", time.Now().Unix())
						go self.buildBlock()
						go self.timeoutSubmission()
					}

					wait := time.Duration(int64(20)) * time.Second
					time.Sleep(wait)
				}
			} else {
				wait := time.Duration(int64(2)) * time.Second
				time.Sleep(wait)
			}

		} else {

			fmt.Println("Space  Block : nil  ")
			wait := time.Duration(int64(10)) * time.Second
			time.Sleep(wait)
		}
	}
}

func (self *Worker) commitNewBlock(validate *protocol.Block) {

	if self.validateBlock(validate, true) {
		self.spaceChain.Insert(validate)
	}
}

func (self *Worker) validateBlock(validate *protocol.Block, update bool) bool {

	transactions := validate.Transactions()
	txs := make(protocol.Txs, len(transactions))
	copy(txs, transactions)
	block := self.spaceChain.CurrentBlock()
	if block != nil {

		//fmt.Println("validate.GetParentHash() : ", validate.GetParentHash(), "Block Hash : ", block.Hash())
		if (validate.GetParentHash() != block.Hash()) || (validate.GetNumberU64() <= (block.GetNumberU64())) {
			return false
		}

		currentstate, _ := state.New(block.GetRoot(), block.GetRootAccount(), state.NewDatabase(self.chainDb))
		spaceid := new(big.Int).Add(block.BlockHead().IdIndex, big.NewInt(0))
		cost := big.NewInt(0)
		walletid := block.WalletIndex()
		for _, tx3 := range txs {

			cost = new(big.Int).Add(cost, tx3.Cost())
			toadd := tx3.To()
			switch ptype := tx3.Data()[0]; ptype {
			case 1:
				walletid = currentstate.CreateSpaceAndBindAddr(spaceid,walletid, *toadd)
				currentstate.SetBalance(*toadd, big.NewInt(0))
				spaceid = new(big.Int).Add(spaceid, big.NewInt(1))
			case 2:
				currentstate.AddBalance(*toadd, tx3.Value())
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			case 3:

				var dataAttrSet protocol.Attr
				if err := rlp.DecodeBytes(tx3.Data()[1:], &dataAttrSet); err != nil {
					fmt.Println("rlp.DecodeBytes failed: ", err)
				}
				currentstate.SetAttribute(*tx3.To(), dataAttrSet.K, dataAttrSet.V)
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			case 4:
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			case 5:
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
				//gwallet.GetGWallet(0).CoinPublish(from, tx3.Value().Uint64())
			case 6:
				fmt.Println("epochTx")
			case 7:
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			case 8:
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			case 9:
				Liquidationdata, _ := self.spaceChain.GetLiquidationReport(tx3.Hash())
				if len(Liquidationdata) > 0 {
					self.LiquidationToState(Liquidationdata, currentstate)
				}
			case 10:
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			case 11:
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			case 12:
				from, _ := protocol.Sender(protocol.FrontierSigner{}, tx3)
				currentstate.SubBalance(from, tx3.Cost())
			default:
				continue
			}
		}

		rootSpace, rootAccount, _ := currentstate.CommitTo(self.chainDb)
		if (validate.GetRoot() == rootSpace) && (validate.GetRootAccount() == rootAccount) {
			//fmt.Println("***********  RootSpace RootAccount == *************** ")
			if (validate.GetSpaceIndexU64() == spaceid.Uint64()) && (validate.GetGasUsedU64() == cost.Uint64()) {
				fmt.Println("************* Validate  Block : true ************** ")
				return true
			}
		}

	}

	return false
}

func (self *Worker) LiquidationToState(data []byte, state *state.StateDB) error {
	var reports protocol.LiquidationReports
	err := rlp.DecodeBytes(data, &reports)
	if err == nil {
		for _, report := range reports {

			amount := big.NewInt(0).SetUint64(report.Gas)
			balance, _ := state.GetSpaceBalanceByID(report.Space_Id)
			if balance.Uint64() > amount.Uint64() {
				state.SubSpaceBalanceByID(report.Space_Id, amount)
			} else {
				state.SubSpaceBalanceByID(report.Space_Id, balance)
			}
		}
	}

	return err
}

func (self *Worker) buildBlock() {

	txmsgs := self.txpool.GetAllTx()
	var LiquidationHash string
	if len(txmsgs) > 0 {

		self.gw = gwallet.GetGWallet(0)
		block := self.spaceChain.CurrentBlock()
		if block != nil {

			currentstate, _ := state.New(block.GetRoot(), block.GetRootAccount(), state.NewDatabase(self.chainDb))
			fmt.Println("Block Number : ", block.GetNumberU64(), "Block Hash : ", block.Hash())

			var txs protocol.Txs
			spaceid := new(big.Int).Add(block.BlockHead().IdIndex, big.NewInt(0))
			cost := big.NewInt(0)
			walletid := block.WalletIndex()
			groupid  := block.GroupIndex()

			for _, txmsg := range txmsgs {

				cost = new(big.Int).Add(cost, txmsg.Txmsg.Cost)
				toadd := txmsg.Txmsg.To
				txs = append(txs, &txmsg.Txs)
				switch ptype := txmsg.Txmsg.Data[0]; ptype {
				case 1:
					walletid = currentstate.CreateSpaceAndBindAddr(spaceid,walletid, *toadd)
					currentstate.SetBalance(*toadd, big.NewInt(0))
					spaceid = new(big.Int).Add(spaceid, big.NewInt(1))
				case 2:
					currentstate.AddBalance(*toadd, txmsg.Txmsg.Amount)
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 3:
					var dataAttrSet protocol.Attr
					if err := rlp.DecodeBytes(txmsg.Txmsg.Data[1:], &dataAttrSet); err != nil {
						fmt.Println("rlp.DecodeBytes failed: ", err)
					}
					currentstate.SetAttribute(*txmsg.Txmsg.To, dataAttrSet.K, dataAttrSet.V)
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 4:
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 5:
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 6:
					fmt.Println("epochTx")
				case 7:
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 8:
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 9:
					LiquidationHash = string(txmsg.Txmsg.Data[1:])
					Liquidationdata, _ := self.spaceChain.GetLiquidationReport(txmsg.Txs.Hash())
					if len(Liquidationdata) > 0 {
						self.LiquidationToState(Liquidationdata, currentstate)
					}
				case 10:
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 11:
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				case 12:
					currentstate.SubBalance(*txmsg.Txmsg.From, txmsg.Txmsg.Cost)
				default:
					continue
				}
			}

			rootSpace, rootAccount, _ := currentstate.CommitTo(self.chainDb)
			groupConstHash := self.validateGroup.Hash()
			//fmt.Println("groupConstHash : ",groupConstHash)
			//fmt.Println("******* state.CommitTo  rootSpace, ParentHash ", rootSpace, rootAccount)

			addrs := self.validateGroup.SelectConsensus(block.GetNumberU64() + 1)
			if len(addrs) >= 3 {

				self.fixed = false
				self.blockVrf = vrf.NewVrf(0, self.sender, addrs)
				self.coefficient, self.sortIndex = self.blockVrf.ValidateBlock(block.GetNumberU64(), block.Hash(), self.sender)
			} else {
				self.coefficient = float32(1.0)
				self.fixed = true
				if len(addrs) > 0 {
					self.fixedAddress = addrs[0]
					fmt.Println("Space Worker fixed block address :  ", self.fixedAddress)
				}
			}

			var ptype byte
			b := new(bytes.Buffer)

			if self.fixed {
				ptype = 1
				b.WriteByte(ptype)
				b.Write(self.sender.Bytes())
			} else {

				self.bls = blsgroup.GetGroup(uint(0))
				pubkey := self.bls.PubKey()
				ptype = 2
				b.WriteByte(ptype)
				b.Write(pubkey)
			}

			head := &protocol.Header{
				Number:          new(big.Int).Add(block.Number(), common.Big1),
				Timestamp:       big.NewInt(time.Now().Unix()),
				UncleHash:       block.Hash(),
				Version:         block.GetVersion(),
				GasUsed:         cost,
				IdIndex:         spaceid,
				WalletID:        walletid,
				GroupID:         groupid,
				SpaceRoot:       rootSpace,
				AccRoot:         rootAccount,
				CoinRoot:        common.Hash{},
				GroupRoot:       common.Hash{},
				ConsensusGroupHash:  groupConstHash,
				LiquidationHash: []byte(LiquidationHash),
				PK:              b.Bytes(),
			}

			tmp := types.Float32ToByte(self.coefficient)
			head.Percent = make([]byte, len(tmp))
			copy(head.Percent, tmp)

			head.PK = make([]byte, len(b.Bytes()))
			copy(head.PK, b.Bytes())
			//fmt.Println("block bls  pubkey:", head.PK)

			body := &protocol.Block_Body{
				Transactions: txs,
			}

			self.currBlock = protocol.NewBlock(head, body)

			hash := self.currBlock.Header.HashNoSig()
			sig, err := self.gw.SignMsg(hash)
			if err != nil {
				fmt.Println("Sig error : ", err)
			}
			//head.Sign = make([]byte, len(sig))
			//copy(head.Sign, sig)
			self.currBlock.SetSigner(sig)

			//fmt.Println("Austin sig hash：", hash.Hex(), "Sig:", self.currBlock.Header.Sign)
			//addre, err := self.gw.SignSender(hash, self.currBlock.Header.Sign)
			//if err != nil {
			//	fmt.Println("Sig error : ", err)
			//	return
			//}
			//fmt.Println("Address : ", self.sender.Hex(), "Sig Address : ", addre.Hex())

			fmt.Println("self.fixedAddress : ", self.fixedAddress, " Address : ", self.sender)
			if self.fixedAddress == self.sender && self.fixed {

				self.blockIsTrue = false
				fmt.Println("self addr:", self.fixedAddress.Hex(), "sender addr:", self.sender.Hex())
				self.validateGroup.SetlastGroupHash(self.currBlock.GetConsensusGroupHash())
				fmt.Println("fixed  Space Block Number : ", self.currBlock.GetNumberU64(), "blockHash : ", self.currBlock.Hash())
				ev := NewBlockEvent{self.currBlock.Hash(), self.currBlock}
				self.newBlockFeed.Send(ev)
				go self.spaceChain.Insert(self.currBlock)

			} else {

				if !self.fixed {

					if self.blockVrf != nil {
						self.coefficient, self.sortIndex = self.blockVrf.ValidateBlock(block.GetNumberU64(), block.Hash(), self.sender)
					} else {
						self.coefficient = float32(1.0)
					}

					if self.blockVrf != nil {
						self.sigBlocks[self.currBlock.Hash()] = self.currBlock
						self.blockIsTrue = true
						if self.sortIndex == 0 {
							self.blockIsTrue = false
							fmt.Println("sender LastBlock addr:", self.sender.Hex())
							ev := LastBlockEvent{self.currBlock.Hash(), self.currBlock}
							self.lastBlockFeed.Send(ev)
							self.lastBlockCh <- ev
							fmt.Println("LastBlock  number : ", self.currBlock.GetNumberU64(), "Hash : ", self.currBlock.Hash())
						}
					}

				}

			}
		}
	}
}

func (self *Worker) SubscribeLastBlockEvent(ch chan<- LastBlockEvent) event.Subscription {
	return self.scope.Track(self.lastBlockFeed.Subscribe(ch))
}

func (self *Worker) SubscribeBlockSigEvent(ch chan<- BlockSigEvent) event.Subscription {
	return self.scope.Track(self.blockSigFeed.Subscribe(ch))
}

func (self *Worker) SubscribeNewBlockEvent(ch chan<- NewBlockEvent) event.Subscription {
	return self.scope.Track(self.newBlockFeed.Subscribe(ch))
}
