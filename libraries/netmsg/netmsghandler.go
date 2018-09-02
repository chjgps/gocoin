package netmsg

import (
	"fmt"
	"sync"
	"sync/atomic"
	"gocoin/libraries/event"
	"gocoin/libraries/rlp"
)

const (
	NetMsg                = 0x01
	NewNetMsgBlock       = 0x02
)

type NetmsgHandler struct {
     netmsgchan  chan  Netmsg
	 stop        chan struct{}

	newNetMsgFeed   event.Feed
	scope           event.SubscriptionScope

	mu      sync.Mutex
	atWork  int32
}

func NewNetmsgHandler() *NetmsgHandler  {

	handler := &NetmsgHandler{
		netmsgchan: make(chan Netmsg,1024),
		stop:       make(chan struct{} ),
	}
	return  handler
}

func (nmh *NetmsgHandler) SubscribeNewBlockEvent(ch  chan<- Netmsg) event.Subscription {
	return nmh.scope.Track(nmh.newNetMsgFeed.Subscribe(ch))
}

func (nmh *NetmsgHandler) Start()  {

	nmh.mu.Lock()
	defer nmh.mu.Unlock()
	if atomic.LoadInt32(&nmh.atWork) == 1 {
		return
	}
	atomic.StoreInt32(&nmh.atWork, 1)

	go nmh.handlerMsgLoop()
}

func (nmh *NetmsgHandler) Stop()  {

	nmh.mu.Lock()
	defer nmh.mu.Unlock()
	if atomic.LoadInt32(&nmh.atWork) == 1 {
		nmh.stop <- struct{}{}
	}
	atomic.StoreInt32(&nmh.atWork, 0)
}

func (nmh *NetmsgHandler) AddLocalNetMsg(msgtype  uint8, val interface{}) error  {
	msgdata,err := rlp.EncodeToBytes(val)

	if err != nil {
		return  err
	}
	netmsg := Netmsg{
		MsgType:msgtype,
		MsgData:msgdata,
	}
	fmt.Println("Add new netmsg : ",netmsg)

	return  nil
}

func (nmh *NetmsgHandler) HandlerMsgchan( msg  Netmsg )  {
	nmh.netmsgchan <- msg
}

func (nmh *NetmsgHandler) handlerMsgLoop()  {

	for {
		select {
		case netmsg :=  <-nmh.netmsgchan:
		  	nmh.hangletmsg(netmsg)
		case <-nmh.stop:
			break
		}
	}
}

func (nmh *NetmsgHandler) hangletmsg(netmsg Netmsg)  {

	if len(netmsg.MsgData) > 0 {
		switch  {
		case netmsg.MsgType == NetMsg:
			{

			}
		case netmsg.MsgType == NewNetMsgBlock:
			{

			}
		case netmsg.MsgType == 0:
			fmt.Println("handleMsg default : ", "netmsg.MsgData", netmsg.MsgData)
		default:
			fmt.Println("handleMsg default : ", "netmsg.MsgType",netmsg.MsgType, "netmsg.MsgData", netmsg.MsgData)
		}
	}
}
