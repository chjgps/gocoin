package common

import (
	"sync"
	"sync/atomic"

	"gocoin/libraries/common/netserver"
	"gocoin/libraries/event"
)

type TcpServerManage struct {
	tcpservermap map[uint32]*netserver.TcpServer
	reveivechan  chan netserver.NewMsgEvent

	newMsgFeed event.Feed
	scope      event.SubscriptionScope

	stop   chan struct{}
	atWork int32
	mu     sync.Mutex
}

func NewTcpServerManage() *TcpServerManage {
	tsm := &TcpServerManage{
		tcpservermap: make(map[uint32]*netserver.TcpServer),
		reveivechan:  make(chan netserver.NewMsgEvent, 256),
		stop:         make(chan struct{}),
	}

	return tsm
}

func (tsm *TcpServerManage) AddTcpServer(msgtype uint32, tcpserver *netserver.TcpServer) {
	tsm.tcpservermap[msgtype] = tcpserver
}

func (tsm *TcpServerManage) SubscribeNewBlockEvent(ch chan<- netserver.NewMsgEvent) event.Subscription {
	return tsm.scope.Track(tsm.newMsgFeed.Subscribe(ch))
}

func (tsm *TcpServerManage) SendMsg(msg netserver.NewMsgEvent) {

	tcpserver, ok := tsm.tcpservermap[msg.Id]
	if ok {
		tcpserver.SendMsg(msg)
	}
}

func (tsm *TcpServerManage) receiveloop() {

	for {
		select {
		case msg := <-tsm.reveivechan:
			{
				tsm.newMsgFeed.Send(msg)
			}
		case <-tsm.stop:
			break
		}
	}
}

func (tsm *TcpServerManage) Start() {

	if len(tsm.tcpservermap) > 0 {

		tsm.mu.Lock()
		defer tsm.mu.Unlock()

		if atomic.LoadInt32(&tsm.atWork) == 1 {
			return
		}

		atomic.StoreInt32(&tsm.atWork, 1)

		for _, tcpserver := range tsm.tcpservermap {
			tcpserver.SubscribeNewBlockEvent(tsm.reveivechan)
			tcpserver.Start()
		}

		go tsm.receiveloop()
	}

}

func (tsm *TcpServerManage) Stop() {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	if atomic.LoadInt32(&tsm.atWork) == 1 {

		for _, tcpserver := range tsm.tcpservermap {
			tcpserver.Stop()
		}
		tsm.stop <- struct{}{}
	}

	atomic.StoreInt32(&tsm.atWork, 0)
}
