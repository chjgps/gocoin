// Copyright(c)2017-2020 gocoin,Co,.Ltd.

package publish

import (
	"bytes"
	"fmt"
	"gocoin/libraries/chain/space"
	"gocoin/libraries/chain/space/protocol"
	trie2 "gocoin/libraries/chain/space/trie"
	"gocoin/libraries/chain/transaction/gwallet"
	"gocoin/libraries/common"
	"gocoin/libraries/crypto/sha3"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/event"
	"gocoin/libraries/rlp"
	"math/big"
	"time"
)

const (
	err_none        = 0xFF
	err_inner_input = 0xFF + 1
	publishNumber   = 65535 // the coins number which will be published
	publishLimited  = 2000  // the min_number for the publish
	publishFinal    = 20000 // the actual number which the coins received
)

type PublishCoins struct {
	Hash    common.Hash
	Name    string
	CoinNum *big.Int
	Owner   *Publisher
}

type Cycle interface {
	SetTime()

	SetBlock()
}

type Limit interface {
	SetLimit()

	SetDamage()

	SetExchange()
}

type LifelineSession struct {
	cyc Cycle
	lim Limit
}

type NewSession struct {
	num       big.Int
	recoinage uint
	identify  byte
	singleMax int32

	// null will be no limited
	location []int16
	scope    []int64

	// lifeline setting
	lf LifelineSession
}

type Scrivener struct {
	Addr common.Address
	Sign []byte
}

type Publisher struct {
	orgindex common.Hash    // Old contract index
	index    common.Hash    // contract index
	addr     common.Address // address
	gasLimit big.Int
	watchers []Scrivener //
	sign     [64]byte
	time     time.Time
	Contract []byte
}

type CoinSession struct {
	id    uint64
	name  [8]byte
	owner Publisher
}

type BlockShort struct {
	bc  *space.BlockChain
	num uint64
}

// GenerateRule for a new block received from other node
func GenerateRule(blks <-chan BlockShort) {
	blk := <-blks
	header := blk.bc.GetHeaderByNumber(blk.num)

	fmt.Println("block head :", header)
	// if the new rule had been published and the event if not null
}

func generateTx(data []byte) (ptx *protocol.Tx) {
	return protocol.NewPublishCon(new(big.Int).SetBytes(data), new(big.Int).SetUint64(0), data)
}

func printDetail(data []byte) {
	// TODO: execut the contract
}

func publishSession(data []byte) (bool, int) {
	if len(data) <= 0 {
		return false, err_inner_input
	}

	cdetail := PublishCoins{}
	err := rlp.DecodeBytes(data, cdetail)

	if err != nil {
		return false, err_inner_input
	}

	cdetail = PublishCoins{
		Hash:    rlpHash(data),
		Name:    "RMB",
		CoinNum: new(big.Int).SetUint64(1000),
		Owner: &Publisher{
			orgindex: common.Hash{},
			index:    rlpHash(data),
			addr:     gwallet.GetGWallet(0).Gaddr,
			gasLimit: *big.NewInt(0),
			watchers: []Scrivener{},
			time:     time.Now(),
			Contract: data,
		},
	}

	// TODO: Get the publish contract

	// Print the contract execute result
	printDetail(data)

	// TODO: Generate the tx
	buf := new(bytes.Buffer)
	rlp.Encode(buf, cdetail)
	ptx := generateTx(buf.Bytes())
	_ = ptx
	// TODO: Send the tx
	return true, err_none
}

// reveive the publish event
func Recv(msg PublishCoins) {
	/*
		datail := msg

		var horg common.Hash
		h := sha3.NewKeccak256()
		rlp.Encode(h, datail.Data)
		h.Sum(horg[:0])

		if datail.Dash != horg {
			return
		}
	*/
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type Publish struct {
	db           *lvldb.LDBDatabase
	trieContract *trie2.Trie
	trieResult   *trie2.Trie
	coinMsgCh    event.Feed
	txCh         chan protocol.TxPreEvent
	txpool       *space.TxPool
	blockChain   *space.BlockChain
	gw           *gwallet.GWallet
}

var lpublish *Publish

func NewPublish(contract lvldb.Database, result lvldb.Database, txpool *space.TxPool, blockChain *space.BlockChain) (*Publish, error) {

	contractTrie, err := trie2.New(common.Hash{}, contract)
	if err != nil {
		return nil, err
	}

	resultTrie, err := trie2.New(common.Hash{}, result)
	if err != nil {
		return nil, err
	}

	txCh := make(chan protocol.TxPreEvent)
	txpool.SubscribeTxPreEvent(txCh)

	pub := &Publish{
		//db:           contract.(*lvldb.LDBDatabase),
		trieContract: contractTrie,
		trieResult:   resultTrie,
		txCh:         txCh,
		txpool:       txpool,
		blockChain:   blockChain,
	}
	lpublish = pub
	return pub, nil
}

type CoinPublishMsg struct {
	PubCoins *PublishCoins
}

type CoinPublishMsgEvent struct{ CoinPublishMsg *CoinPublishMsg }

func (self *Publish) PublishSession(data []byte, addr common.Address) (bool, int) {
	if len(data) <= 0 {
		return false, err_inner_input
	}

	notary := GetNotary()
	req := notary.ReqNotary(addr, []byte("Hello World"))
	res := notary.WaitNotary(req.CttHash)

	for _, v := range res.Notary {
		fmt.Println("Notary:", v.Addr.Hex())
	}

	cdetail := &PublishCoins{
		Hash:    rlpHash(data),
		Name:    "RMB",
		CoinNum: new(big.Int).SetUint64(1000),
		Owner: &Publisher{
			orgindex: common.Hash{},
			index:    rlpHash(data),
			addr:     gwallet.GetGWallet(0).Gaddr,
			gasLimit: *big.NewInt(0),
			watchers: []Scrivener{},
			time:     time.Now(),
			Contract: data,
		},
	}

	self.coinMsgCh.Send(CoinPublishMsgEvent{&CoinPublishMsg{cdetail}})
	return true, err_none
}

func (self *Publish) Recv(msg *PublishCoins) error {
	buf := new(bytes.Buffer)
	err := rlp.Encode(buf, msg)
	if err != nil {
		fmt.Printf("err msg %v: %v", msg, err)
		return err
	}
	tx := protocol.NewPublishCon(new(big.Int).SetUint64(1000), new(big.Int).SetUint64(0), buf.Bytes())
	siger := protocol.MakeSigner()
	tx, err = protocol.SignTx(tx, siger, gwallet.GetGWallet(0).Gprv)
	if err != nil {
		fmt.Println(err)
	}
	err = self.txpool.AddLocal(tx)
	if err != nil {
		fmt.Println("Recv functoin:", err)
	}
	return nil
}

func (self *Publish) Worker() {
	for {
		select {
		case event := <-self.txCh:
			fmt.Println("Publish requesst")
			tx := event.Transaction
			data := tx.Data()
			if data[0] != protocol.IdPublishCoin {
				continue
			}
			signer := protocol.MakeSigner()
			from, err := protocol.Sender(signer, tx)
			_ = from
			if err != nil {
				continue
			}

			/*
			   var pos []uint64
			   var coinnum uint64

			   if self.blockChain.CurrentBlock() != nil {
			       coinnum = self.blockChain.CurrentBlock().GetCoinNum()
			   } else {
			       coinnum = 10000
			   }

			   coinnum = 10000
			   for i := uint64(0); i < tx.Value().Uint64(); i++ {
			       pos = append(pos, coinnum+i)
			   }
			   msg := CoinPublishMsg{
			       Owner:    from,
			       Position: pos,
			   }
			   msg.Hash = rlpHash([]interface{}{msg.Owner, msg.Position})
			   //gwallet.UpdateGwByCoinpub(msg.Owner, msg.Position)
			*/
		}
	}
}

func (self *Publish) SubCoinPublishEvent(ch chan CoinPublishMsgEvent) {
	self.coinMsgCh.Subscribe(ch)
}

func (self *Publish) PubCoin(add common.Address, coin uint64) {

}

func GetPublish() *Publish {
	return lpublish
}
