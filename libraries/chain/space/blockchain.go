/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : block chain
**/

package space

import (
	"errors"
	"fmt"
	"gocoin/libraries/chain/blsgroup"
	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/chain/space/state"
	"gocoin/libraries/chain/transaction/gwallet"
	"gocoin/libraries/chain/validategroup"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/event"
	"gocoin/libraries/log"
	"gocoin/libraries/rlp"
	"math/big"
	"strings"
	"sync"
)

var ErrNoGenesis = errors.New("Genesis not found in chain")

const (
	Epoch = 3
)

type BlockChain struct {
	hc           *HeaderChain
	chainDb      lvldb.Database
	dataDir      string
	genesisBlock *protocol.Block
	currentBlock *protocol.Block

	stateCache state.Database // State database to reuse between imports (contains state cache)

	validateGroup *validategroup.ValidateGroup

	createthresholdFeed event.Feed
	scope               event.SubscriptionScope

	mu      sync.RWMutex
	chainmu sync.RWMutex // blockchain insertion lock

	wg sync.WaitGroup // chain processing wait group for shutting down

}

func NewBlockChain(chainDb lvldb.Database, dataDir string) (*BlockChain, error) {
	bc := &BlockChain{
		chainDb:    chainDb,
		dataDir:    dataDir,
		stateCache: state.NewDatabase(chainDb),
	}

	var err error
	bc.hc, err = NewHeaderChain(chainDb)
	if err != nil {
		return nil, err
	}

	bc.genesisBlock = bc.GetBlockByNumber(0)

	if bc.genesisBlock != nil {
		bc.currentBlock = bc.genesisBlock
	} else {

		bc.genesisBlock, _ = GetGenesisBlock(chainDb, dataDir)
		if bc.genesisBlock == nil {
			return nil, ErrNoGenesis
		} else {
			bc.currentBlock = bc.genesisBlock
		}
	}

	if err := bc.loadLastState(); err != nil {
		return nil, err
	}

	return bc, nil
}

func (bc *BlockChain) SetValidateGroup(validateGroup *validategroup.ValidateGroup) {
	bc.validateGroup = validateGroup
}

func (bc *BlockChain) SubscribeCreatethresholdEvent(ch chan<- blsgroup.CreatethresholdEvent) event.Subscription {
	return bc.scope.Track(bc.createthresholdFeed.Subscribe(ch))
}

func (bc *BlockChain) BlockSigerVerifier(block *protocol.Block) bool {

	if len(block.Header.Sign) > 0 {
		switch ptype := block.GetPK()[0]; ptype {
		case 1:
			{
				fmt.Println("block.Header.HashNoSig:", block.Header.HashNoSig().Hex(), "Sig:", block.Header.Sign)
				addre, err := gwallet.GetGWallet(0).SignSender(block.Header.HashNoSig(), block.Header.Sign)
				if err != nil {
					fmt.Println("Sig error : ", err)
					return false
				}
				fmt.Println("Address : ", block.GetPK()[1:], "Sig Address : ", addre.Hex())
				str := string(block.GetPK()[1:])
				straddr := addre.Str()
				if strings.Compare(str, straddr) == 0 {
					return true
				}
			}
		case 2:
			{
				hash := block.Header.HashNoSig()
				fmt.Println("block.Header.HashNoSig:", hash.Hex(), "Sig:", block.Header.Sign)
				if blsgroup.Verify(block.Header.Sign, hash.Bytes(), block.GetPK()[1:]) {
					return true
				}
			}
		}

		return false
	}

	return false
}

func (bc *BlockChain) Process(block *protocol.Block, statedb *state.StateDB) {

	transactions := block.Transactions()
	txs := make(protocol.Txs, len(transactions))
	copy(txs, transactions)

	costsum := big.NewInt(0)
	spaceid := bc.currentBlock.SpaceIndex()
	walletindex := bc.currentBlock.WalletIndex()
	groupindex := bc.currentBlock.GroupIndex()
	for _, tx := range txs {
		spaceid, walletindex,groupindex, costsum = bc.ApplyTransaction(spaceid,costsum, walletindex,groupindex, tx, statedb)
	}

	number := block.GetNumberU64()
	if number%Epoch == 0 && number != 0 {
		if bc.validateGroup != nil {

			//space  consensus
			bc.validateGroup.Reset(block.GetConsensusGroupHash(), false)
			if bc.validateGroup.GetThresholdSize() >= 3 {
				addrs := bc.validateGroup.SelectThresholds(number)
				fmt.Println("Bls space addrs", len(addrs))
				ev := blsgroup.CreatethresholdEvent{0, protocol.RlpHash(addrs), addrs}
				bc.createthresholdFeed.Send(ev)
			}

			//transaction coin
			bc.validateGroup.ResetCoinGroup()
			if bc.validateGroup.GetThresholdCoinSize() >= 3 {
				addrs := bc.validateGroup.SelectCoinThresholds(number)
				fmt.Println("Bls coin validate addrs", len(addrs))
				ev := blsgroup.CreatethresholdEvent{1, protocol.RlpHash(addrs), addrs}
				bc.createthresholdFeed.Send(ev)
			}

			//transaction  consensus
			bc.validateGroup.ResetTxGroup()
			if bc.validateGroup.GetThresholdTxSize() >= 3 {
				addrs := bc.validateGroup.SelectTxThresholds(number)
				fmt.Println("Bls Tx consensus addrs", len(addrs))
				ev := blsgroup.CreatethresholdEvent{2, protocol.RlpHash(addrs), addrs}
				bc.createthresholdFeed.Send(ev)
			}
		}
	}
}

func (bc *BlockChain) LiquidationToState(data []byte, state *state.StateDB) error {
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

func (bc *BlockChain) ApplyTransaction(idIndex , cost *big.Int, wIndex uint64, gIndex uint32, tx *protocol.Tx, currentstate *state.StateDB) (*big.Int, uint64,uint32, *big.Int) {

	spaceid := new(big.Int).Add(idIndex, big.NewInt(0))
	walletid := wIndex
	groupid  := gIndex
	costsum := new(big.Int).Add(cost, tx.Cost())

	toadd := tx.To()
	switch ptype := tx.Data()[0]; ptype {
	case 1:
		walletid =  currentstate.CreateSpaceAndBindAddr(spaceid,walletid, *toadd)
		currentstate.SetBalance(*toadd, big.NewInt(0))
		spaceid = new(big.Int).Add(spaceid, big.NewInt(1))
	case 2:
		currentstate.AddBalance(*toadd, tx.Value())
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
	case 3:
		var dataAttrSet protocol.Attr
		if err := rlp.DecodeBytes(tx.Data()[1:], &dataAttrSet); err != nil {
			fmt.Println("rlp.DecodeBytes failed: ", err)
		}
		currentstate.SetAttribute(*tx.To(), dataAttrSet.K, dataAttrSet.V)
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
	case 4:
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
	case 5:
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
		fmt.Println("CoinPublish  Address : " ,from.Hex()," Coin : ",tx.Value().Uint64())

	case 6:
		fmt.Println("epochTx")
	case 7:
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
	case 8:
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
		if bc.validateGroup != nil {
			groupuser := validategroup.GroupMember{
				Address: from,
				Time:    big.NewInt(int64(tx.Time())),
			}
			bc.validateGroup.Add(groupuser)
		}
	case 9:
		Liquidationdata, _ := bc.GetLiquidationReport(tx.Hash())
		if len(Liquidationdata) > 0 {
			bc.LiquidationToState(Liquidationdata, currentstate)
			bc.DeleteLiquidationReport(tx.Hash())

		}
	case 10:
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
		if bc.validateGroup != nil {
			groupuser := validategroup.GroupMember{
				Address: from,
				Time:    big.NewInt(int64(tx.Time())),
			}
			bc.validateGroup.AddStell(groupuser)
		}
	case 11:
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
		if bc.validateGroup != nil {
			groupuser := validategroup.GroupMember{
				Address: from,
				Time:    big.NewInt(int64(tx.Time())),
			}
			bc.validateGroup.AddCoin(groupuser)
		}
	case 12:
		from, _ := protocol.Sender(protocol.FrontierSigner{}, tx)
		currentstate.SubBalance(from, tx.Cost())
		if bc.validateGroup != nil {
			groupuser := validategroup.GroupMember{
				Address: from,
				Time:    big.NewInt(int64(tx.Time())),
			}
			bc.validateGroup.AddTransaction(groupuser)
		}
	}

	return spaceid, walletid, groupid, costsum
}

func (bc *BlockChain) Insert(block *protocol.Block) {

	if bc.BlockSigerVerifier(block) {

		fmt.Println("block number  : ", block.GetNumberU64(), "currentBlock : ", block)

		if block.GetNumberU64() == bc.currentBlock.GetNumberU64() {
			log.Debug("block number  ", "new block: ", block.GetNumberU64(), "currentBlock : ", bc.currentBlock.GetNumberU64())
			return
		}

		updateHeads := GetCanonicalHash(bc.chainDb, block.GetNumberU64()) != block.Hash()
		// If the block is better than out head or is on a different chain, force update heads
		if updateHeads {

			currentstate, _ := state.New(bc.currentBlock.GetRoot(), bc.currentBlock.GetRootAccount(), state.NewDatabase(bc.chainDb))
			if currentstate != nil {

				bc.Process(block, currentstate)

				rootSpace, rootAccount, _ := currentstate.CommitTo(bc.chainDb)
				fmt.Println("block.Root : ", block.GetRoot(), "rootSpace : ", rootSpace, "block.RootAccount", block.GetRootAccount(), "rootAccount : ", rootAccount)
				// Add the block to the canonical chain number scheme and mark as the head
				if err := WriteCanonicalHash(bc.chainDb, block.Hash(), block.GetNumberU64()); err != nil {
					log.Crit("Failed to insert block number", "err", err)
				}
				if err := WriteHeadBlockHash(bc.chainDb, block.Hash()); err != nil {
					log.Crit("Failed to insert head block hash", "err", err)
				}
				bc.currentBlock = block

				bc.hc.SetCurrentHeader(block.BlockHead())

				if err := WriteBlock(bc.chainDb, block); err != nil {
					log.Crit("Failed to Write block  to db ", "err", err)
				}
			}
		}

	} else {
		fmt.Println("Siger Verifier Failed to insert block number ", block.GetNumberU64())
	}
}

// insertChain will execute the actual chain insertion and event aggregation. The
// only reason this method exists as a separate one is to make locking cleaner
// with deferred statements.
func (bc *BlockChain) insertChain(chain protocol.Blocks) (int, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].GetNumberU64() != chain[i-1].GetNumberU64()+1 || chain[i].GetParentHash() != chain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			log.Error("Non contiguous block insert", "number", chain[i].GetNumberU64(), "hash", chain[i].Hash(),
				"parent", chain[i].GetParentHash(), "prevnumber", chain[i-1].GetNumberU64(), "prevhash", chain[i-1].Hash())

			return 0, nil
		}
	}
	// Pre-checks passed, start the full block imports
	bc.wg.Add(1)
	defer bc.wg.Done()

	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	// Iterate over the blocks and insert when the verifier permits
	for _, block := range chain {
		bc.Insert(block)
	}

	return len(chain), nil
}

// GetBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
func (bc *BlockChain) GetBlocksFromHash(hash common.Hash, n int) (blocks []*protocol.Block) {
	number := bc.hc.GetBlockNumber(hash)
	for i := 0; i < n; i++ {
		block := bc.GetBlock(hash, number)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		hash = block.GetParentHash()
		number--
	}

	return
}

func (bc *BlockChain) loadLastState() error {

	head := GetHeadBlockHash(bc.chainDb)
	if head == (common.Hash{}) {
		log.Warn("Empty database, resetting chain")
		return nil
	}

	currentBlock := bc.GetBlockByHash(head)
	if currentBlock == nil {
		log.Warn("Head block missing, resetting chain", "hash", head)
		return nil
	}

	bc.currentBlock = currentBlock

	currentHeader := bc.currentBlock.BlockHead()
	if head := GetHeadHeaderHash(bc.chainDb); head != (common.Hash{}) {
		if header := bc.GetHeaderByHash(head); header != nil {
			currentHeader = header
		}
	}
	bc.hc.SetCurrentHeader(currentHeader)

	return nil
}

func (bc *BlockChain) GetHeaderByHash(hash common.Hash) *protocol.Header {
	return bc.hc.GetHeaderByHash(hash)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (bc *BlockChain) GetHeaderByNumber(number uint64) *protocol.Header {
	return bc.hc.GetHeaderByNumber(number)
}

func (bc *BlockChain) LastBlockHash() common.Hash {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.currentBlock.Hash()
}

func (bc *BlockChain) CurrentBlock() *protocol.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.currentBlock

}

func (bc *BlockChain) Genesis() *protocol.Block {

	return bc.genesisBlock
}

func (bc *BlockChain) GetBody(hash common.Hash) *protocol.Block_Body {
	body := GetBody(bc.chainDb, hash, bc.hc.GetBlockNumber(hash))
	if body == nil {
		return nil
	}

	return body

}

func (bc *BlockChain) GetBodyRLP(hash common.Hash) rlp.RawValue {
	body := GetBodyRLP(bc.chainDb, hash, bc.hc.GetBlockNumber(hash))
	if len(body) == 0 {
		return nil
	}

	return body
}

func (bc *BlockChain) GetBlock(hash common.Hash, number uint64) *protocol.Block {
	block := GetBlock(bc.chainDb, hash, number)
	if block == nil {
		return nil
	}

	return block
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *protocol.Block {

	return bc.GetBlock(hash, bc.hc.GetBlockNumber(hash))
}

func (bc *BlockChain) GetBlockByNumber(number uint64) *protocol.Block {

	hash := GetCanonicalHash(bc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return bc.GetBlock(hash, number)
}

// HasBlock checks if a block is fully present in the database or not.
func (bc *BlockChain) HasBlock(hash common.Hash, number uint64) bool {
	ok, _ := bc.chainDb.Has(blockBodyKey(hash, number))
	return ok
}

func (bc *BlockChain) Status() (currentBlock common.Hash, genesisBlock common.Hash) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	log.Debug("block chain status", "currentBlock ", bc.currentBlock, "genesisBlock", bc.genesisBlock)
	if bc.currentBlock == nil && bc.genesisBlock == nil {
		return common.Hash{}, common.Hash{}
	}

	if bc.currentBlock == nil && bc.genesisBlock != nil {
		return common.Hash{}, bc.genesisBlock.Hash()
	}

	if bc.currentBlock != nil && bc.genesisBlock == nil {
		return bc.currentBlock.Hash(), common.Hash{}
	}

	return bc.currentBlock.Hash(), bc.genesisBlock.Hash()
}
func (bc *BlockChain) CurrentState(rootSpace, rootAccount common.Hash) (*state.StateDB, error) {
	return state.New(rootSpace, rootAccount, bc.stateCache)
}

func (bc *BlockChain) IsStellMember(addr common.Address) bool {
	if bc.validateGroup != nil {
		return bc.validateGroup.IsStellAddress(addr)
	}
	return false
}

func (bc *BlockChain) GetStellGroupSize() uint64 {
	return bc.validateGroup.GetStellGroupSize()
}
func (bc *BlockChain) GetStellGroup() validategroup.GroupMembers {
	return bc.validateGroup.GetStellGroup()
}

func (bc *BlockChain) SaveLiquidationReport(datahash common.Hash, data []byte) error {
	return WriteLiquidationHash(bc.chainDb, datahash, data)
}

func (bc *BlockChain) GetLiquidationReport(datahash common.Hash) ([]byte, error) {
	return GetLiquidation(bc.chainDb, datahash)
}

func (bc *BlockChain) DeleteLiquidationReport(datahash common.Hash) error {
	return DeleteLiquidationHash(bc.chainDb, datahash)
}

/*
func (bc *BlockChain) GetTxChainSettle(hash common.Hash) *settle.Body {
	// TODO: add the settle block to space chain body
	var data settle.Body
	return data
}
*/
