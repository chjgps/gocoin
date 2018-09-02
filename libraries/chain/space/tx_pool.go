/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : transaction pool
**/
package space

import (
	"errors"
	"fmt"
	"sync"

	"bytes"
	"gocoin/libraries/chain/space/epoch"
	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/chain/space/state"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/event"
	"gocoin/libraries/ipfs"
	"gocoin/libraries/log"
	"gocoin/libraries/rlp"
	"os"
)

const (
/* channel for listenning recv chain block header */
//chainHeaderListen = 10

/* channel for listenning recv remove-tx messages */
//removeTxListen = 10
)

var (
	ErrInvalidSender      = errors.New("invalid sender")
	ErrReplaceUnderpriced = errors.New("replacement transaction underpriced")
	ErrNegativeValue      = errors.New("negative value")
	ErrOversizedData      = errors.New("oversized data")
	ErrCostLimit          = errors.New("exceeds block Cost limit")
	ErrInsufficientFunds  = errors.New("insufficient funds for Cost + value")
)

type blockChain interface {
	CurrentBlock() *protocol.Block
	GetBlock(hash common.Hash, blocknum uint64) *protocol.Block
	CurrentState(rootSpace common.Hash, rootAccount common.Hash) (*state.StateDB, error)
	SaveLiquidationReport(datahash common.Hash, data []byte) error
	IsStellMember(addr common.Address) bool
}

type Block_Tx struct {
	Txmsg protocol.Message
	Txs   protocol.Tx
	Stell map[common.Hash][]byte
}
type BlockTxPreEvent struct{ BlockTx *Block_Tx }

type TxPool struct {
	chain          blockChain
	chainDb        lvldb.Database
	currentMaxCost uint64
	signer         protocol.Signer
	rwLock         sync.RWMutex

	epoch *epoch.Epoch

	currentState *state.StateDB // Current state in the blockchain head
	txpending    map[common.Address]*txList
	txqueue      map[common.Address]*txList
	all          map[common.Hash]*protocol.Tx

	txFeed event.Feed
	btFeed event.Feed

	wg sync.WaitGroup
}

func NewTxPool(chain blockChain, spacechainDb lvldb.Database) *TxPool {

	pool := &TxPool{
		chain:          chain,
		chainDb:        spacechainDb,
		signer:         protocol.MakeSigner(),
		currentMaxCost: 10000,
		txpending:      make(map[common.Address]*txList),
		txqueue:        make(map[common.Address]*txList),
		all:            make(map[common.Hash]*protocol.Tx),
	}

	return pool
}

func (pool *TxPool) Stop() {

	pool.wg.Wait()
	log.Info("Transaction pool stopped")
}

// SubscribeTxPreEvent registers a subscription of core.TxPreEvent and
// starts sending event to the given channel.
func (pool *TxPool) SubscribeTxPreEvent(ch chan protocol.TxPreEvent) {
	pool.txFeed.Subscribe(ch)
}

func (pool *TxPool) SetEpoch(epoch *epoch.Epoch) {
	pool.epoch = epoch
}

func (pool *TxPool) SubscribeBlockTxPreEvent(ch chan BlockTxPreEvent) {
	pool.btFeed.Subscribe(ch)
}

func (pool *TxPool) Content() (map[common.Address]protocol.Txs, map[common.Address]protocol.Txs) {
	pool.rwLock.Lock()
	defer pool.rwLock.Unlock()

	pending := make(map[common.Address]protocol.Txs)
	for addr, list := range pool.txpending {
		pending[addr] = list.Flatten()
	}
	queued := make(map[common.Address]protocol.Txs)
	for addr, list := range pool.txqueue {
		queued[addr] = list.Flatten()
	}
	return pending, queued
}

func (pool *TxPool) Pending() (map[common.Address]protocol.Txs, error) {
	pool.rwLock.Lock()
	defer pool.rwLock.Unlock()

	pending := make(map[common.Address]protocol.Txs)
	for addr, list := range pool.txpending {
		pending[addr] = list.Flatten()
	}
	return pending, nil
}

func (pool *TxPool) UpdateState(newHead *protocol.Header) {
	pool.rwLock.Lock()
	defer pool.rwLock.Unlock()

	pool.currentState, _ = pool.chain.CurrentState(newHead.SpaceRoot, newHead.AccRoot)
}

func (pool *TxPool) validateTx(tx *protocol.Tx) (common.Address, error) {
	if tx.Size() > 32*1024 {
		return common.Address{}, ErrOversizedData
	}

	if tx.Value().Sign() < 0 {
		return common.Address{}, ErrNegativeValue
	}

	// todo
	if tx.Cost().Uint64() > pool.currentMaxCost {
		return common.Address{}, ErrCostLimit
	}

	from, err := protocol.Sender(pool.signer, tx)
	if err != nil {
		return common.Address{}, ErrInvalidSender
	}

	//if pool.currentState.GetBalance(from).Uint64() < (tx.Cost().Uint64()+tx.Value().Uint64()) {
	//	return common.Address{} ,ErrInsufficientFunds
	//}

	block := pool.chain.CurrentBlock()
	if block != nil {
		currentState, err := state.New(block.GetRoot(), block.GetRootAccount(), state.NewDatabase(pool.chainDb))
		if err != nil {
			return common.Address{}, ErrInvalidSender
		}
		balance, err := currentState.GetBalance(from)
		fmt.Println("statHash:", block.GetRoot(), "Account:", block.GetRootAccount())
		fmt.Println("Blance:", balance.Uint64())
		if err != nil {
			return common.Address{}, err
		}
		if balance.Uint64() < (tx.Cost().Uint64() + tx.Value().Uint64()) {
			return common.Address{}, ErrInsufficientFunds
		}
	}
	return from, nil
}

func (pool *TxPool) add(tx *protocol.Tx, local bool) (bool, error) {

	hash := tx.Hash()
	if pool.all[hash] != nil {
		return false, fmt.Errorf("known transaction: %x", hash)
	}

	from, err := pool.validateTx(tx)
	if err != nil {
		return false, err
	}

	switch ptype := tx.Data()[0]; ptype {
	case 6:
		{
			if pool.epoch != nil {
				pool.epoch.AddTx(tx)
			}
			return true, nil
		}
	}

	replace, err := pool.enqueueTx(from, hash, tx)
	if err != nil {
		return false, err
	}

	//log.Trace("Pooled new future transaction", "hash", hash, "from", from, "to", tx.To())
	fmt.Println("Austin Pooled new future transaction", "hash", hash, "from", from, "to", tx.To())
	pool.txFeed.Send(protocol.TxPreEvent{tx})
	fmt.Println("Austin have send txfeed")

	return replace, nil
}

type report struct {
	Hash common.Hash
	Sign [][]byte
}

func (r *report) Sender(hash common.Hash, sign []byte) common.Address {
	return common.Address{}
}

func (pool *TxPool) downLoad(tx *protocol.Tx) {
	KENLEN := 46
	key := string(tx.Data()[1 : 1+KENLEN])
	if err := ipfs.Get(key, "/tmp"); err != nil {
		return
	}

	file, err := os.Open("/tmp/" + key)
	if err != nil {
		return
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	if err != nil {
		return
	}

	key = string(tx.Data()[1+KENLEN:])
	if err := ipfs.Get(key, "/tmp"); err != nil {
		return
	}
	file, err = os.Open("/tmp/" + key)
	if err != nil {
		return
	}
	reportbuf := new(bytes.Buffer)
	_, err = reportbuf.ReadFrom(file)
	if err != nil {
		return
	}
	s := rlp.NewStream(reportbuf, 0)
	var sport report
	s.Decode(&sport)
	for _, v := range sport.Sign {
		addr := sport.Sender(sport.Hash, v)
		if pool.chain.IsStellMember(addr) == false {
			return
		}
	}

	pool.rwLock.RLock()
	defer pool.rwLock.RUnlock()
	pool.chain.SaveLiquidationReport(tx.Hash(), buf.Bytes())
	pool.all[tx.Hash()] = tx
}

func (pool *TxPool) enqueueTx(addr common.Address, hash common.Hash, tx *protocol.Tx) (bool, error) {
	ptype := tx.Data()[0]
	if ptype == protocol.IdStellReport {
		go pool.downLoad(tx)
	} else {
		pool.all[hash] = tx
	}
	return false, nil
}

func (pool *TxPool) promoteTx(addr common.Address, hash common.Hash, tx *protocol.Tx) {

	if pool.txpending[addr] == nil {
		pool.txpending[addr] = newTxList()
	}
	list := pool.txpending[addr]

	inserted, old := list.Add(tx)
	if !inserted {

		delete(pool.all, hash)
		return
	}

	if old != nil {
		delete(pool.all, old.Hash())
	}

	if pool.all[hash] == nil {
		pool.all[hash] = tx
	}

	// todo go pool.txFeed.Send(TxPreEvent{tx})
	pool.txFeed.Send(protocol.TxPreEvent{tx})

}

func (pool *TxPool) AddLocal(tx *protocol.Tx) error {
	return pool.addTx(tx, true)
}

func (pool *TxPool) AddRemote(tx *protocol.Tx) error {
	return pool.addTx(tx, false)
}

func (pool *TxPool) AddLocals(txs []*protocol.Tx) []error {
	return pool.addTxs(txs, true)
}

func (pool *TxPool) AddRemotes(txs []*protocol.Tx) []error {
	return pool.addTxs(txs, false)
}

func (pool *TxPool) addTx(tx *protocol.Tx, local bool) error {
	pool.rwLock.Lock()
	defer pool.rwLock.Unlock()

	_, err := pool.add(tx, local)
	if err != nil {
		return err
	}

	return nil
}

func (pool *TxPool) addTxs(txs []*protocol.Tx, local bool) []error {
	pool.rwLock.Lock()
	defer pool.rwLock.Unlock()

	return pool.addTxsLocked(txs, local)
}

func (pool *TxPool) addTxsLocked(txs []*protocol.Tx, local bool) []error {

	dirty := make(map[common.Address]struct{})
	errs := make([]error, len(txs))

	for i, tx := range txs {
		var replace bool
		if replace, errs[i] = pool.add(tx, local); errs[i] == nil {
			if !replace {
				from, _ := protocol.Sender(pool.signer, tx)
				dirty[from] = struct{}{}
			}
		}
	}

	if len(dirty) > 0 {
		addrs := make([]common.Address, 0, len(dirty))
		for addr := range dirty {
			addrs = append(addrs, addr)
		}
		pool.promoteExecutables(addrs)
	}
	return errs
}

func (pool *TxPool) Get(hash common.Hash) *protocol.Tx {
	pool.rwLock.RLock()
	defer pool.rwLock.RUnlock()

	return pool.all[hash]
}

func (pool *TxPool) removeTx(hash common.Hash) {

	tx, ok := pool.all[hash]
	if !ok {
		return
	}
	addr, _ := protocol.Sender(pool.signer, tx)

	delete(pool.all, hash)

	// Remove the transaction from the pending lists and reset the account nonce
	if pending := pool.txpending[addr]; pending != nil {
		if removed, invalids := pending.Remove(tx); removed {
			// If no more transactions are left, remove the list
			if pending.Empty() {
				delete(pool.txpending, addr)
			} else {
				// Otherwise postpone any invalidated transactions
				for _, tx := range invalids {
					pool.enqueueTx(addr, tx.Hash(), tx)
				}
			}
			return
		}
	}

	if future := pool.txqueue[addr]; future != nil {
		future.Remove(tx)
		if future.Empty() {
			delete(pool.txqueue, addr)
		}
	}
}

func (pool *TxPool) promoteExecutables(accounts []common.Address) {
	if accounts == nil {
		accounts = make([]common.Address, 0, len(pool.txqueue))
		for addr := range pool.txqueue {
			accounts = append(accounts, addr)
		}
	}

	for _, addr := range accounts {
		list := pool.txqueue[addr]
		if list == nil {
			continue
		}
		//todo moves transactions that have become processable from the future queue to the set of pending transactions.

		for _, tx := range list.Flatten() {
			hash := tx.Hash()
			log.Trace("Promoting queued transaction", "hash", hash)
			pool.promoteTx(addr, hash, tx)
		}

		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(pool.txqueue, addr)
		}
	}
}

func (pool *TxPool) demoteUnexecutables() {
	// Iterate over all accounts and demote any non-executable transactions
	for addr, list := range pool.txpending {

		//todo removes invalid and processed transactions from the pools

		if list.Empty() {
			delete(pool.txpending, addr)
		}
	}
}

func (pool *TxPool) GetTxsCount() int {
	pool.rwLock.RLock()
	defer pool.rwLock.RUnlock()
	return len(pool.all)
}

func (pool *TxPool) GetAllTx() []*Block_Tx {
	pool.rwLock.RLock()
	defer pool.rwLock.RUnlock()

	txmsgs := []*Block_Tx{}
	for _, tx := range pool.all {

		h := tx.Hash()
		from, _ := protocol.Sender(pool.signer, tx)
		msg := protocol.Message{
			Hash:   &h,
			To:     tx.To(),
			From:   &from,
			Cost:   tx.Cost(),
			Amount: tx.Value(),
			Data:   tx.Data(),
		}
		txmsg := &Block_Tx{msg, *tx, nil}
		txmsgs = append(txmsgs, txmsg)
	}
	return txmsgs
}

func (pool *TxPool) RemoveTx(hash []common.Hash) {
	pool.rwLock.RLock()
	defer pool.rwLock.RUnlock()
	for _, v := range hash {
		delete(pool.all, v)
	}
}
