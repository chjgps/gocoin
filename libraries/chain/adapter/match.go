// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// The file used to check the rules and local
// data matched result
package adapter

import (
	"bytes"
	"fmt"
	"gocoin/libraries/chain/publish"
	"gocoin/libraries/chain/space"
	"gocoin/libraries/chain/space/protocol"
	trie2 "gocoin/libraries/chain/space/trie"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/event"
	"gocoin/libraries/rlp"
	"math/big"
	"time"
)

var (
	rulePath string = ""
	ruleDB   *lvldb.LDBDatabase
)

type ruleStatus struct {
	num    int32
	latest common.Hash
}

type ruleReport struct {
	orgNum          int32
	lostOrginList   []common.Hash
	lostExecuteList []common.Hash
}

func SetConfig(path string) {
	rulePath = path
}

func loadRuleDb(addr common.Address) bool {
	var defaultPath string = ""
	if 0 == len(rulePath) {
		defaultPath = fmt.Sprintf("./%s", addr.Hex())
	} else {
		defaultPath = rulePath
	}

	var err error
	ruleDB, err = lvldb.NewLDBDatabase(defaultPath, 2, 1)
	if err != nil {
		fmt.Println("create the rule database failed.")
		return false
	}

	return true
}

func matchRules() (rr ruleReport) {
	return rr
}

func getWalletRules(latestHash common.Hash) bool {
	// TODO: get the newest rule database

	// TODO: read wallet include coin sessions

	// TODO: create the new rules report

	return true
}

func fetchRules(rr ruleReport) bool {
	return false
}

// try to keep a dynamic same contract version
func SyncRule(bc *space.BlockChain) bool {
	// Read the latest block header
	curhash := bc.CurrentBlock().Header().RootPublish

	// load the local rule database
	var addr common.Address
	if !loadRuleDb(addr) {
		fmt.Println("SyncRule: failed.")
	}

	// get the current rule
	if !getWalletRules(curhash) {
		fmt.Println("local rules is old.")
	}

	// fetch the support rules

	fmt.Println(curhash)

	return true
}

func saveRule(src RuleSource) {

}

/****************************************/
type RuleSet struct {
	db           *lvldb.LDBDatabase
	ContractTrie *trie2.Trie
	ResultTrie   *trie2.Trie
	CTTFeed      event.Feed
}

func NewRuleSet(contract lvldb.Database, result lvldb.Database) (*RuleSet, error) {
	contractTrie, err := trie2.New(common.Hash{}, contract)
	if err != nil {
		return nil, err
	}

	resultTrie, err := trie2.New(common.Hash{}, result)
	if err != nil {
		return nil, err
	}

	ruleSet := &RuleSet{
		db:           contract.(*lvldb.LDBDatabase),
		ContractTrie: contractTrie,
		ResultTrie:   resultTrie,
	}
	return ruleSet, nil
}

func (rset *RuleSet) saveRule(rule []byte, result []byte) error {
	h := protocol.RlpHash(rule)
	rsrc := RuleSource{
		K:    h,
		V:    0,
		VSrc: rule,
	}
	rexe := RuleExecute{
		K:     h,
		VTime: time.Time{},
		VSrc:  result,
	}
	buf := new(bytes.Buffer)
	rlp.Encode(buf, rsrc)
	rset.ContractTrie.Update(h[:], buf.Bytes())
	buf.Reset()
	rlp.Encode(buf, rexe)
	rset.ResultTrie.Update(h[:], buf.Bytes())

	if _, err := rset.ContractTrie.Commit(); err != nil {
		rset.ContractTrie.Delete(h[:])
		return err
	}
	if _, err := rset.ResultTrie.Commit(); err != nil {
		rset.ResultTrie.Delete(h[:])
		return err
	}
	return nil
}

func (rset *RuleSet) analysisPublishTx(tx *protocol.Tx) error {

	signer := protocol.MakeSigner()
	_, err := protocol.Sender(signer, tx)
	if err != nil {
		return err
	}

	data := tx.Data()
	if data[0] != protocol.IdPublishCoin {
		return fmt.Errorf("Not publish Tx")
	}

	data = data[1:]
	buf := bytes.NewBuffer(data)
	var pub publish.PublishCoins
	if err := rlp.Decode(buf, &pub); err != nil {
		return err
	}

	Contract := pub.Owner.Contract
	result := []byte{} //execute Contract
	rset.saveRule(Contract, result)
	return nil
}

func (rset *RuleSet) Subscribe(ev chan ContractSyncMsgEvent) {
	rset.CTTFeed.Subscribe(ev)
}

const (
	idReqCTT = 0x01
	idResCTT = 0x02
)

type ContractSyncMsg struct {
	MsgType  int
	Hash     common.Hash
	CTTHash  common.Hash
	Contract []byte
}
type ContractSyncMsgEvent struct{ ev *ContractSyncMsg }

func (rset *RuleSet) syncRule(hash common.Hash) {
	ctt := rset.ContractTrie.Get(hash[:])
	if ctt != nil {
		return
	}
	ev := &ContractSyncMsg{
		MsgType:  idReqCTT,
		Hash:     common.Hash{},
		CTTHash:  hash,
		Contract: nil,
	}
	rset.CTTFeed.Send(ContractSyncMsgEvent{ev})
}

func (rset *RuleSet) RecvRule(msg *ContractSyncMsg) {
	if msg.MsgType == idResCTT {
		result := []byte{} //execute contract
		rset.saveRule(msg.Contract, result)
	}

	if msg.MsgType == idReqCTT {
		contract := rset.ContractTrie.Get(msg.CTTHash[:])
		msg := &ContractSyncMsg{
			MsgType:  idResCTT,
			Hash:     common.Hash{},
			CTTHash:  msg.CTTHash,
			Contract: contract,
		}
		rset.CTTFeed.Send(ContractSyncMsgEvent{msg})
	}
}

// RPC interface for get the special coin session rule
func GetSpecRule(id big.Int) (rc *RecoinageCase) {
	return rc
}
