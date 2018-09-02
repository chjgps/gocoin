/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : transaction pool test
**/

package space

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/chain/space/state"
	"gocoin/libraries/common"
	//"gocoin/libraries/crypto"
	"bytes"
	"io/ioutil"
	"os"
)

type testBlockChain struct {
}

func (bc *testBlockChain) CurrentBlock() *protocol.Block {
	return &protocol.Block{}
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *protocol.Block {
	return bc.CurrentBlock()
}

func (bc *testBlockChain) CurrentState(common.Hash) (*state.StateDB, error) {
	return &state.StateDB{}, nil
}

func transaction(gascost *big.Int, key *ecdsa.PrivateKey) *protocol.Tx {
	return pricedTransaction(gascost, key)
}

func pricedTransaction(gascost *big.Int, key *ecdsa.PrivateKey) *protocol.Tx {
	tx, _ := protocol.SignTx(protocol.NewTx(common.Address{}, big.NewInt(100), gascost, nil), protocol.FrontierSigner{}, key)
	return tx
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	// db, _ := ethdb.NewMemDatabase()
	// statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	// blockchain := &testBlockChain{statedb, big.NewInt(1000000), new(event.Feed)}

	//key, _ := crypto.GenerateKey()
	//pool := NewTxPool(&testBlockChain{})

	//return pool, key
	return nil, nil
}

func deriveSender(tx *protocol.Tx) (common.Address, error) {
	return protocol.Sender(protocol.FrontierSigner{}, tx)
}

func TestTransactions(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(big.NewInt(100), key)
	if err := pool.AddRemote(tx); err != ErrReplaceUnderpriced {
		t.Error("expected", ErrReplaceUnderpriced, "got", err)
	}

	tx = transaction(big.NewInt(100000), key)
	if err := pool.AddRemote(tx); err != ErrReplaceUnderpriced {
		t.Error("expected", ErrReplaceUnderpriced, "got", err)
	}
	if err := pool.AddLocal(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}
}

func TestBuf(t *testing.T) {
	file, err := os.Open("/tmp/ipfsfile")
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	if err != nil {
		return
	}
	file.Seek(0, 0)
	b, _ := ioutil.ReadAll(file)
	s := string(b)
	t.Log(s)
}
