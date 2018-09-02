/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2018.5.4
 * Website    : http:www.gocoin.com
 * Function   : Complaint transaction in epoch
**/
package epoch

import (
	"fmt"
	"sync"
	"sync/atomic"

	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/chain/transaction/gwallet"
	"gocoin/libraries/event"
)

type Epoch struct {
	txchan    chan *protocol.Tx
	epochFeed event.Feed
	scope     event.SubscriptionScope

	stop   chan struct{}
	mu     sync.Mutex
	atWork int32
}

func NewEpoch() *Epoch {
	epoch := &Epoch{
		txchan: make(chan *protocol.Tx, 32),
		stop:   make(chan struct{}),
	}

	return epoch
}

func (ep *Epoch) SubscribeEpochEvent(ch chan<- *protocol.Tx) event.Subscription {
	return ep.scope.Track(ep.epochFeed.Subscribe(ch))
}

func (ep *Epoch) AddTx(tx *protocol.Tx) {
	ep.txchan <- tx
}

func (ep *Epoch) Start() {

	ep.mu.Lock()
	defer ep.mu.Unlock()

	if atomic.LoadInt32(&ep.atWork) == 1 {
		return
	}

	atomic.StoreInt32(&ep.atWork, 1)

	go ep.VerificationEpochTx()
}

func (ep *Epoch) Stop() {

	ep.mu.Lock()
	defer ep.mu.Unlock()
	if atomic.LoadInt32(&ep.atWork) == 1 {
		ep.stop <- struct{}{}
	}

	atomic.StoreInt32(&ep.atWork, 0)
}

func (ep *Epoch) VerificationEpochTx() {
	for {
		select {
		case tx := <-ep.txchan:
			{
				ep.commitTxpool(tx)
			}
		case <-ep.stop:
			{
				return
			}
		}
	}
}

func (ep *Epoch) commitTxpool(tx *protocol.Tx) {

	hash := tx.DataHash()
	sig, err := gwallet.GetGWallet(0).SignMsg(hash)
	if err != nil {
		fmt.Println("Sig error : ", err)
		return
	}

	tx.AddEpochSig(sig)
	ep.epochFeed.Send(tx)
}
