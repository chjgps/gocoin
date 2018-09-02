/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : Sort the TXs and ready for processing
**/
package space

import (
	"gocoin/libraries/chain/space/protocol"
	"sort"
)

type txList struct {
	txs protocol.Txs
}

// type txSortedMap struct {
// 	txItems map[uint64]*protocol.Tx
// }

// func newTxSortedMap() *txSortedMap {
// 	return &txSortedMap{
// 		txItems: protocol.Txs,
// 	}
// }

// func (m *txSortedMap) Get(index uint64) *protocol.Tx {
// 	return m.txItems[index]
// }

// func (m *txSortedMap) Put(tx *protocol.Tx) {
// 	m.txItems = append(m.txItems, tx)
// }

// // Remove deletes a transaction from the maintained map, returning whether the
// // transaction was found.
// func (m *txSortedMap) Remove(idx uint64) bool {

// 	if idx < len(m.txItems) {
// 		m.txItems = append(m.txItems[:index], m.txItems[index+1:]...)
// 		return false
// 	}

// 	return true
// }

// // Ready retrieves a sequentially increasing list of transactions starting at the
// // provided nonce that is ready for processing. The returned transactions will be
// // removed from the list.
// //
// // Note, all transactions with nonces lower than start will also be returned to
// // prevent getting into and invalid state. This is not something that should ever
// // happen but better to be self correcting than failing!
// func (m *txSortedMap) Ready(start uint64) protocol.Txs {
// 	// Short circuit if no transactions are available
// 	if m.index.Len() == 0 || (*m.index)[0] > start {
// 		return nil
// 	}
// 	// Otherwise start accumulating incremental transactions
// 	var ready protocol.Txs
// 	for next := (*m.index)[0]; m.index.Len() > 0 && (*m.index)[0] == next; next++ {
// 		ready = append(ready, m.txItems[next])
// 		delete(m.items, next)
// 		heap.Pop(m.index)
// 	}
// 	m.cache = nil

// 	return ready
// }

// func (m *txSortedMap) Flatten() protocol.Txs {
// 	txs := make(protocol.Txs, len(m.items))
// 	copy(txs, m.items)
// 	sort.Sort(protocol.TxByCost(txs))
// 	return txs
// }

func newTxList() *txList {
	return &txList{}
}

func (l *txList) Flatten() protocol.Txs {
	result := make(protocol.Txs, len(l.txs))
	copy(result, l.txs)
	sort.Sort(protocol.TxByCost(result))
	return result
}

func (l *txList) Add(tx *protocol.Tx) (bool, *protocol.Tx) {
	l.txs = append(l.txs, tx)
	return true, nil
}

func (l *txList) Remove(tx *protocol.Tx) (bool, protocol.Txs) {
	index := -1
	for i, t := range l.txs {
		if t.Hash() == tx.Hash() {
			index = i
		}
	}

	if index > -1 {
		l.txs = append(l.txs[:index], l.txs[index+1:]...)
		return true, nil
	}

	return false, nil
}

func (l *txList) Len() int {
	return l.txs.Len()
}

func (l *txList) Empty() bool {
	return l.Len() == 0
}
