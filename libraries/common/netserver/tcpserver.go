package netserver

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"gocoin/libraries/event"
	"gocoin/libraries/rlp"
)

type NewMsgEvent struct {
	Id      uint32
	MsgData []byte
}

type TcpServer struct {
	id      uint32
	network string
	address string
	conn    net.Conn

	newMsgFeed event.Feed
	scope      event.SubscriptionScope
	sendMsgCh  chan NewMsgEvent

	connected bool

	stop     chan struct{}
	netclose bool
	atWork   int32
	mu       sync.Mutex
}

func chkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func NewTcpServer(id uint32, network, address string) *TcpServer {
	tcp := &TcpServer{
		id:        id,
		network:   network,
		address:   address,
		sendMsgCh: make(chan NewMsgEvent, 32),
		stop:      make(chan struct{}),
	}

	tcp.netclose = false
	return tcp
}

func (tcp *TcpServer) Start() {

	tcp.mu.Lock()
	defer tcp.mu.Unlock()

	if atomic.LoadInt32(&tcp.atWork) == 1 {
		return
	}

	atomic.StoreInt32(&tcp.atWork, 1)

	go tcp.listenLoop()
	go tcp.sendMsgServer()
}

func (tcp *TcpServer) Stop() {

	tcp.mu.Lock()
	defer tcp.mu.Unlock()

	if atomic.LoadInt32(&tcp.atWork) == 1 {
		tcp.stop <- struct{}{}
		tcp.netclose = true
		tcp.conn.Close()
	}

	atomic.StoreInt32(&tcp.atWork, 0)
}

func (tcp *TcpServer) listenLoop() {

	tcpaddr, err := net.ResolveTCPAddr(tcp.network, tcp.address)
	chkError(err)
	tcplisten, err1 := net.ListenTCP(tcp.network, tcpaddr)
	chkError(err1)
	for {
		conn, err2 := tcplisten.Accept()
		if err2 != nil {
			chkError(err2)

			if tcp.netclose {
				break
			}

			continue
		}

		if !tcp.connected {
			tcp.connected = true
			var data [4]byte
			binary.BigEndian.PutUint32(data[:], tcp.id)
			tcp.conn = conn
			tcp.conn.Write(data[:])

			go tcp.clientHandle(tcp.conn)
		}

	}
}

func (tcp *TcpServer) SendMsg(msg NewMsgEvent) {

	if tcp.conn != nil && tcp.connected {
		tcp.sendMsgCh <- msg
	}

}

func (tcp *TcpServer) sendMsgServer() {
	for {
		select {
		case msg := <-tcp.sendMsgCh:
			{
				data, err := rlp.EncodeToBytes(msg)
				if err != nil {
					chkError(err)
					continue
				}
				fmt.Println("Send client data")
				n, err := tcp.conn.Write(data)
				fmt.Println("Send client data len : ", n, "err : ", err)
			}
		case <-tcp.stop:
			return
		}
	}
}

func (tcp *TcpServer) clientHandle(conn net.Conn) {

	defer conn.Close()
	for {

		data := make([]byte, 4096)
		n, err := conn.Read(data)
		if n == 0 || err != nil {
			chkError(err)
			continue
		}

		msgbuffer := data[0:n]

		fmt.Println("conn.Read : ", msgbuffer)
		var msg NewMsgEvent
		if err := rlp.DecodeBytes(msgbuffer, &msg); err != nil {
			chkError(err)
			continue
		}
		fmt.Println("msg ID : ", msg.Id, "msg data : ", msg.MsgData)
		tcp.newMsgFeed.Send(msg)
	}
}

func (tcp *TcpServer) SubscribeNewBlockEvent(ch chan<- NewMsgEvent) event.Subscription {
	return tcp.scope.Track(tcp.newMsgFeed.Subscribe(ch))
}
