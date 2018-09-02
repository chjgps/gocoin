package shim

import (
	"errors"
	"sync"
	"fmt"
	pb "gocoin/libraries/chaincodecore/protos/peer"
	"github.com/looplab/fsm"
	"github.com/golang/protobuf/proto"
)


// PeerChaincodeStream interface for stream between Peer and chaincode instance.
type PeerChaincodeStream interface {
	Send(*pb.ChaincodeMessage) error
	Recv() (*pb.ChaincodeMessage, error)
	CloseSend() error
}

type nextStateInfo struct {
	msg      *pb.ChaincodeMessage
	sendToCC bool
}

func (handler *Handler) triggerNextState(msg *pb.ChaincodeMessage, send bool) {
	handler.nextState <- &nextStateInfo{msg, send}
}


// Handler handler implementation for shim side of chaincode.
type Handler struct {
	sync.RWMutex
	//shim to peer grpc serializer. User only in serialSend
	serialLock sync.Mutex
	To         string
	ChatStream PeerChaincodeStream
	FSM        *fsm.FSM
	cc         Chaincode
	// Multiple queries (and one transaction) with different txids can be executing in parallel for this chaincode
	// responseChannel is the channel on which responses are communicated by the shim to the chaincodeStub.
	responseChannel map[string]chan pb.ChaincodeMessage
	nextState       chan *nextStateInfo
}


func shorttxid(txid string) string {
	if len(txid) < 8 {
		return txid
	}
	return txid[0:8]
}

//serialSend serializes msgs so gRPC will be happy
func (handler *Handler) serialSend(msg *pb.ChaincodeMessage) error {
	handler.serialLock.Lock()
	defer handler.serialLock.Unlock()

	err := handler.ChatStream.Send(msg)

	return err
}

// NewChaincodeHandler returns a new instance of the shim side handler.
func newChaincodeHandler(peerChatStream PeerChaincodeStream, chaincode Chaincode) *Handler {
	v := &Handler{
		ChatStream: peerChatStream,
		cc:         chaincode,
	}
	v.responseChannel = make(map[string]chan pb.ChaincodeMessage)
	v.nextState = make(chan *nextStateInfo)

	// Create the shim side FSM
	v.FSM = fsm.NewFSM(
		"created",
		fsm.Events{
			{Name: pb.ChaincodeMessage_REGISTERED.String(), Src: []string{"created"}, Dst: "established"},
			{Name: pb.ChaincodeMessage_READY.String(), Src: []string{"established"}, Dst: "ready"},
			{Name: pb.ChaincodeMessage_ERROR.String(), Src: []string{"init"}, Dst: "established"},
			{Name: pb.ChaincodeMessage_RESPONSE.String(), Src: []string{"init"}, Dst: "init"},
			{Name: pb.ChaincodeMessage_INIT.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pb.ChaincodeMessage_TRANSACTION.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pb.ChaincodeMessage_RESPONSE.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pb.ChaincodeMessage_ERROR.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pb.ChaincodeMessage_COMPLETED.String(), Src: []string{"init"}, Dst: "ready"},
			{Name: pb.ChaincodeMessage_COMPLETED.String(), Src: []string{"ready"}, Dst: "ready"},
		},
		fsm.Callbacks{
			"before_" + pb.ChaincodeMessage_REGISTERED.String():  func(e *fsm.Event) { v.beforeRegistered(e) },
			"after_" + pb.ChaincodeMessage_RESPONSE.String():     func(e *fsm.Event) { v.afterResponse(e) },
			"after_" + pb.ChaincodeMessage_ERROR.String():        func(e *fsm.Event) { v.afterError(e) },
			"before_" + pb.ChaincodeMessage_INIT.String():        func(e *fsm.Event) { v.beforeInit(e) },
			"before_" + pb.ChaincodeMessage_TRANSACTION.String(): func(e *fsm.Event) { v.beforeTransaction(e) },
		},
	)
	return v
}

// beforeRegistered is called to handle the REGISTERED message.
func (handler *Handler) beforeRegistered(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.ChaincodeMessage); !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	chaincodeLogger.Debug("Received %s, ready for invocations", pb.ChaincodeMessage_REGISTERED)
}

// handleInit handles request to initialize chaincode.
func (handler *Handler) handleInit(msg *pb.ChaincodeMessage) {
	// The defer followed by triggering a go routine dance is needed to ensure that the previous state transition
	// is completed before the next one is triggered. The previous state transition is deemed complete only when
	// the beforeInit function is exited. Interesting bug fix!!
	go func() {
		var nextStateMsg *pb.ChaincodeMessage

		send := true

		defer func() {
			handler.triggerNextState(nextStateMsg, send)
		}()

		errFunc := func(err error, payload []byte,  errFmt string, args ...string) *pb.ChaincodeMessage {
			if err != nil {
				// Send ERROR message to chaincode support and change state
				if payload == nil {
					payload = []byte(err.Error())
				}
				chaincodeLogger.Error(errFmt, args)
				return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: payload, Txid: msg.Txid}
			}
			return nil
		}
		// Get the function and args from Payload
		input := &pb.ChaincodeInput{}
		unmarshalErr := proto.Unmarshal(msg.Payload, input)
		if nextStateMsg = errFunc(unmarshalErr, nil,  "[%s]Incorrect payload format. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Call chaincode's Run
		// Create the ChaincodeStub which the chaincode can use to callback
		stub := new(ChaincodeStub)
		err := stub.init(handler, msg.Txid, input)
		if nextStateMsg = errFunc(err, nil, "[%s]Init get error response [%s]. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		res := handler.cc.Init(stub)
		chaincodeLogger.Debug("[%s]Init get response status: %d", shorttxid(msg.Txid), res.Status)

		if res.Status >= ERROR {
			err = fmt.Errorf("%s", res.Message)
			if nextStateMsg = errFunc(err, []byte(res.Message),  "[%s]Init get error response [%s]. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
				return
			}
		}

		resBytes, err := proto.Marshal(&res)
		if nextStateMsg = errFunc(err, nil,  "[%s]Init marshal response error [%s]. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Send COMPLETED message to chaincode support and change state
		nextStateMsg = &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: resBytes, Txid: msg.Txid}
		chaincodeLogger.Debug("[%s]Init succeeded. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_COMPLETED)
	}()
}

// beforeInit will initialize the chaincode if entering init from established.
func (handler *Handler) beforeInit(e *fsm.Event) {
	chaincodeLogger.Debug("Entered state %s", handler.FSM.Current())
	msg, ok := e.Args[0].(*pb.ChaincodeMessage)
	if !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	chaincodeLogger.Debug("[%s]Received %s, initializing chaincode", shorttxid(msg.Txid), msg.Type.String())
	if msg.Type.String() == pb.ChaincodeMessage_INIT.String() {
		// Call the chaincode's Run function to initialize
		handler.handleInit(msg)
	}
}

// handleTransaction Handles request to execute a transaction.
func (handler *Handler) handleTransaction(msg *pb.ChaincodeMessage) {
	// The defer followed by triggering a go routine dance is needed to ensure that the previous state transition
	// is completed before the next one is triggered. The previous state transition is deemed complete only when
	// the beforeInit function is exited. Interesting bug fix!!
	go func() {
		//better not be nil
		var nextStateMsg *pb.ChaincodeMessage

		send := true

		defer func() {
			handler.triggerNextState(nextStateMsg, send)
		}()

		errFunc := func(err error, errStr string, args ...string) *pb.ChaincodeMessage {
			if err != nil {
				payload := []byte(err.Error())
				chaincodeLogger.Error(errStr, args)
				return &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: payload, Txid: msg.Txid}
			}
			return nil
		}

		// Get the function and args from Payload
		input := &pb.ChaincodeInput{}
		unmarshalErr := proto.Unmarshal(msg.Payload, input)
		if nextStateMsg = errFunc(unmarshalErr,  "[%s]Incorrect payload format. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Call chaincode's Run
		// Create the ChaincodeStub which the chaincode can use to callback
		stub := new(ChaincodeStub)
		err := stub.init(handler, msg.Txid, input)
		if nextStateMsg = errFunc(err,  "[%s]Transaction execution failed. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		res := handler.cc.Invoke(stub)

		// Endorser will handle error contained in Response.
		resBytes, err := proto.Marshal(&res)
		if nextStateMsg = errFunc(err,  "[%s]Transaction execution failed. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Send COMPLETED message to chaincode support and change state
		chaincodeLogger.Debug("[%s]Transaction completed. Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_COMPLETED)
		nextStateMsg = &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_COMPLETED, Payload: resBytes, Txid: msg.Txid}
	}()
}

// beforeTransaction will execute chaincode's Run if coming from a TRANSACTION event.
func (handler *Handler) beforeTransaction(e *fsm.Event) {
	msg, ok := e.Args[0].(*pb.ChaincodeMessage)
	if !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}
	chaincodeLogger.Debug("[%s]Received %s, invoking transaction on chaincode(Src:%s, Dst:%s)", shorttxid(msg.Txid), msg.Type.String(), e.Src, e.Dst)
	if msg.Type.String() == pb.ChaincodeMessage_TRANSACTION.String() {
		// Call the chaincode's Run function to invoke transaction
		handler.handleTransaction(msg)
	}
}

// afterResponse is called to deliver a response or error to the chaincode stub.
func (handler *Handler) afterResponse(e *fsm.Event) {
	msg, ok := e.Args[0].(*pb.ChaincodeMessage)
	if !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}

	if err := handler.sendChannel(msg); err != nil {
		chaincodeLogger.Error("[%s]error sending %s (state:%s): %s", shorttxid(msg.Txid), msg.Type, handler.FSM.Current(), err)
	} else {
		chaincodeLogger.Debug("[%s]Received %s, communicated (state:%s)", shorttxid(msg.Txid), msg.Type, handler.FSM.Current())
	}
}

func (handler *Handler) afterError(e *fsm.Event) {
	msg, ok := e.Args[0].(*pb.ChaincodeMessage)
	if !ok {
		e.Cancel(fmt.Errorf("Received unexpected message type"))
		return
	}

	/* TODO- revisit. This may no longer be needed with the serialized/streamlined messaging model
	 * There are two situations in which the ERROR event can be triggered:
	 * 1. When an error is encountered within handleInit or handleTransaction - some issue at the chaincode side; In this case there will be no responseChannel and the message has been sent to the validator.
	 * 2. The chaincode has initiated a request (get/put/del state) to the validator and is expecting a response on the responseChannel; If ERROR is received from validator, this needs to be notified on the responseChannel.
	 */
	if err := handler.sendChannel(msg); err == nil {
		chaincodeLogger.Debug("[%s]Error received from validator %s, communicated(state:%s)", shorttxid(msg.Txid), msg.Type, handler.FSM.Current())
	}
}

func (handler *Handler) sendChannel(msg *pb.ChaincodeMessage) error {
	handler.Lock()
	defer handler.Unlock()
	if handler.responseChannel == nil {
		return fmt.Errorf("[%s]Cannot send message response channel", shorttxid(msg.Txid))
	}
	if handler.responseChannel[msg.Txid] == nil {
		return fmt.Errorf("[%s]sendChannel does not exist", shorttxid(msg.Txid))
	}

	chaincodeLogger.Debug("[%s]before send", shorttxid(msg.Txid))
	handler.responseChannel[msg.Txid] <- *msg
	chaincodeLogger.Debug("[%s]after send", shorttxid(msg.Txid))

	return nil
}

// handleMessage message handles loop for shim side of chaincode/validator stream.
func (handler *Handler) handleMessage(msg *pb.ChaincodeMessage) error {
	if msg.Type == pb.ChaincodeMessage_KEEPALIVE {
		// Received a keep alive message, we don't do anything with it for now
		// and it does not touch the state machine
		return nil
	}
	chaincodeLogger.Debug("[%s]Handling ChaincodeMessage of type: %s(state:%s)", shorttxid(msg.Txid), msg.Type, handler.FSM.Current())
	if handler.FSM.Cannot(msg.Type.String()) {
		err := errors.New(fmt.Sprintf("[%s]Chaincode handler FSM cannot handle message (%s) with payload size (%d) while in state: %s", msg.Txid, msg.Type.String(), len(msg.Payload), handler.FSM.Current()))
		handler.serialSend(&pb.ChaincodeMessage{Type: pb.ChaincodeMessage_ERROR, Payload: []byte(err.Error()), Txid: msg.Txid})
		return err
	}
	err := handler.FSM.Event(msg.Type.String(), msg)
	return filterError(err)
}

// Filter the Errors to allow NoTransitionError and CanceledError to not propagate for cases where embedded Err == nil
func filterError(errFromFSMEvent error) error {
	if errFromFSMEvent != nil {
		if noTransitionErr, ok := errFromFSMEvent.(*fsm.NoTransitionError); ok {
			if noTransitionErr.Err != nil {
				// Squash the NoTransitionError
				return errFromFSMEvent
			}
			chaincodeLogger.Debug("Ignoring NoTransitionError: %s", noTransitionErr)
		}
		if canceledErr, ok := errFromFSMEvent.(*fsm.CanceledError); ok {
			if canceledErr.Err != nil {
				// Squash the CanceledError
				return canceledErr
			}
			chaincodeLogger.Debug("Ignoring CanceledError: %s", canceledErr)
		}
	}
	return nil
}

func (handler *Handler) notify(msg *pb.ChaincodeMessage) {
	//zhangtaowei
	/*handler.Lock()
	defer handler.Unlock()
	tctx := handler.txCtxs[msg.Txid]
	if tctx == nil {
		chaincodeLogger.Debug("notifier Txid:%s does not exist", msg.Txid)
	} else {
		chaincodeLogger.Debug("notifying Txid:%s", msg.Txid)
		tctx.responseNotifier <- msg

		// clean up queryIteratorMap
		for _, v := range tctx.queryIteratorMap {
			v.Close()
		}
	}*/
}

//serialSendAsync serves the same purpose as serialSend (serializ msgs so gRPC will
//be happy). In addition, it is also asynchronous so send-remoterecv--localrecv loop
//can be nonblocking. Only errors need to be handled and these are handled by
//communication on supplied error channel. A typical use will be a non-blocking or
//nil channel
func (handler *Handler) serialSendAsync(msg *pb.ChaincodeMessage, errc chan error) {
	go func() {
		err := handler.serialSend(msg)
		if errc != nil {
			errc <- err
		}
	}()
}

// handleInvokeChaincode communicates with the validator to invoke another chaincode.
func (handler *Handler) handleInvokeChaincode(chaincodeName string, args [][]byte, txid string) pb.Response {
	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: chaincodeName}, Input: &pb.ChaincodeInput{Args: args}})

	// Create the channel on which to communicate the response from validating peer
	var respChan chan pb.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return handler.createResponse(ERROR, []byte(err.Error()))
	}

	defer handler.deleteChannel(txid)

	// Send INVOKE_CHAINCODE message to validator chaincode support
	msg := &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_INVOKE_CHAINCODE, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debug("[%s]Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_INVOKE_CHAINCODE)

	var responseMsg pb.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return handler.createResponse(ERROR, []byte(fmt.Sprintf("[%s]error sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_INVOKE_CHAINCODE)))
	}

	if responseMsg.Type.String() == pb.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debug("[%s]Received %s. Successfully invoked chaincode", shorttxid(responseMsg.Txid), pb.ChaincodeMessage_RESPONSE)
		respMsg := &pb.ChaincodeMessage{}
		if err = proto.Unmarshal(responseMsg.Payload, respMsg); err != nil {
			return handler.createResponse(ERROR, []byte(err.Error()))
		}
		if respMsg.Type == pb.ChaincodeMessage_COMPLETED {
			// Success response
			chaincodeLogger.Debug("[%s]Received %s. Successfully invoed chaincode", shorttxid(responseMsg.Txid), pb.ChaincodeMessage_RESPONSE)
			res := &pb.Response{}
			if err = proto.Unmarshal(respMsg.Payload, res); err != nil {
				return handler.createResponse(ERROR, []byte(err.Error()))
			}
			return *res
		}
		chaincodeLogger.Error("[%s]Received %s. Error from chaincode", shorttxid(responseMsg.Txid), respMsg.Type.String())
		return handler.createResponse(ERROR, responseMsg.Payload)
	}
	if responseMsg.Type.String() == pb.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Error("[%s]Received %s.", shorttxid(responseMsg.Txid), pb.ChaincodeMessage_ERROR)
		return handler.createResponse(ERROR, responseMsg.Payload)
	}

	// Incorrect chaincode message received
	return handler.createResponse(ERROR, []byte(fmt.Sprintf("[%s]Incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pb.ChaincodeMessage_RESPONSE, pb.ChaincodeMessage_ERROR)))
}

func (handler *Handler) createChannel(txid string) (chan pb.ChaincodeMessage, error) {
	handler.Lock()
	defer handler.Unlock()
	if handler.responseChannel == nil {
		return nil, fmt.Errorf("[%s]Cannot create response channel", shorttxid(txid))
	}
	if handler.responseChannel[txid] != nil {
		return nil, fmt.Errorf("[%s]Channel exists", shorttxid(txid))
	}
	c := make(chan pb.ChaincodeMessage)
	handler.responseChannel[txid] = c
	return c, nil
}

func (handler *Handler) createResponse(status int32, payload []byte) pb.Response {
	return pb.Response{Status: status, Payload: payload}
}

func (handler *Handler) deleteChannel(txid string) {
	handler.Lock()
	defer handler.Unlock()
	if handler.responseChannel != nil {
		delete(handler.responseChannel, txid)
	}
}

// handlePutState communicates with the validator to put state information into the ledger.
func (handler *Handler) handlePutState(key string, value []byte, txid string) error {
	// Check if this is a transaction
	chaincodeLogger.Debug("[%s]Inside putstate", shorttxid(txid))

	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pb.PutStateInfo{Key: key, Value: value})

	// Create the channel on which to communicate the response from validating peer
	var respChan chan pb.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return err
	}

	defer handler.deleteChannel(txid)

	// Send PUT_STATE message to validator chaincode support
	msg := &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_PUT_STATE, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debug("[%s]Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_PUT_STATE)

	var responseMsg pb.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return errors.New(fmt.Sprintf("[%s]error sending PUT_STATE %s", msg.Txid, err))
	}

	if responseMsg.Type.String() == pb.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debug("[%s]Received %s. Successfully updated state", shorttxid(responseMsg.Txid), pb.ChaincodeMessage_RESPONSE)
		return nil
	}

	if responseMsg.Type.String() == pb.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Error("[%s]Received %s. Payload: %s", shorttxid(responseMsg.Txid), pb.ChaincodeMessage_ERROR, responseMsg.Payload)
		return errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return errors.New(fmt.Sprintf("[%s]Incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pb.ChaincodeMessage_RESPONSE, pb.ChaincodeMessage_ERROR))
}

//sends a message and selects
func (handler *Handler) sendReceive(msg *pb.ChaincodeMessage, c chan pb.ChaincodeMessage) (pb.ChaincodeMessage, error) {
	errc := make(chan error, 1)
	handler.serialSendAsync(msg, errc)

	//the serialsend above will send an err or nil
	//the select filters that first error(or nil)
	//and continues to wait for the response
	//it is possible that the response triggers first
	//in which case the errc obviously worked and is
	//ignored
	for {
		select {
		case err := <-errc:
			if err == nil {
				continue
			}
			//would have been logged, return false
			return pb.ChaincodeMessage{}, err
		case outmsg, val := <-c:
			if !val {
				return pb.ChaincodeMessage{}, fmt.Errorf("unexpected failure on receive")
			}
			return outmsg, nil
		}
	}
}

// TODO: Implement method to get and put entire state map and not one key at a time?
// handleGetState communicates with the validator to fetch the requested state information from the ledger.
func (handler *Handler) handleGetState(key string, txid string) ([]byte, error) {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pb.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return nil, err
	}

	defer handler.deleteChannel(txid)

	// Send GET_STATE message to validator chaincode support
	payload := []byte(key)
	msg := &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_GET_STATE, Payload: payload, Txid: txid}
	chaincodeLogger.Debug("[%s]Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_GET_STATE)

	var responseMsg pb.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return nil, errors.New(fmt.Sprintf("[%s]error sending GET_STATE %s", shorttxid(txid), err))
	}

	if responseMsg.Type.String() == pb.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debug("[%s]GetState received payload %s", shorttxid(responseMsg.Txid), pb.ChaincodeMessage_RESPONSE)
		return responseMsg.Payload, nil
	}
	if responseMsg.Type.String() == pb.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Error("[%s]GetState received error %s", shorttxid(responseMsg.Txid), pb.ChaincodeMessage_ERROR)
		return nil, errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return nil, errors.New(fmt.Sprintf("[%s]Incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pb.ChaincodeMessage_RESPONSE, pb.ChaincodeMessage_ERROR))
}

// handleDelState communicates with the validator to delete a key from the state in the ledger.
func (handler *Handler) handleDelState(key string, txid string) error {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pb.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return err
	}

	defer handler.deleteChannel(txid)

	// Send DEL_STATE message to validator chaincode support
	payload := []byte(key)
	msg := &pb.ChaincodeMessage{Type: pb.ChaincodeMessage_DEL_STATE, Payload: payload, Txid: txid}
	chaincodeLogger.Debug("[%s]Sending %s", shorttxid(msg.Txid), pb.ChaincodeMessage_DEL_STATE)

	var responseMsg pb.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return errors.New(fmt.Sprintf("[%s]error sending DEL_STATE %s", shorttxid(msg.Txid), pb.ChaincodeMessage_DEL_STATE))
	}

	if responseMsg.Type.String() == pb.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debug("[%s]Received %s. Successfully deleted state", msg.Txid, pb.ChaincodeMessage_RESPONSE)
		return nil
	}
	if responseMsg.Type.String() == pb.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Error("[%s]Received %s. Payload: %s", msg.Txid, pb.ChaincodeMessage_ERROR, responseMsg.Payload)
		return errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return errors.New(fmt.Sprintf("[%s]Incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pb.ChaincodeMessage_RESPONSE, pb.ChaincodeMessage_ERROR))
}
