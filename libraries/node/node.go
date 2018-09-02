// Copyright(C)2017-2020 gocoin,Co,.Ltd.
// All Right Reserved.
//
// This file is part of the go-storage library.

package node

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"

	"gocoin/libraries/chain/account"
	"gocoin/libraries/event"
	"gocoin/libraries/log"
	"gocoin/libraries/net/p2p"
	"gocoin/libraries/rpc"
)

type Node struct {
	eventmux *event.TypeMux // Event multiplexer used between the services of a stack
	accman   *accounts.Manager

	ephemeralKeystore string
	config            *Config
	serverConfig      p2p.Config
	server            *p2p.Server

	serviceFuncs []ServiceConstructor     // Service constructors (in dependency order)
	services     map[reflect.Type]Service // Currently running services

	rpcAPIs       []rpc.API   // List of APIs currently provided by the node
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests

	ipcEndpoint string       // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener // IPC RPC listener socket to serve API requests
	ipcHandler  *rpc.Server  // IPC RPC request handler to process the API requests

	httpEndpoint  string       // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpWhitelist []string     // HTTP RPC modules to allow through this endpoint
	httpListener  net.Listener // HTTP RPC listener socket to server API requests
	httpHandler   *rpc.Server  // HTTP RPC request handler to process the API requests

	wsEndpoint string       // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsListener net.Listener // Websocket RPC listener socket to server API requests
	wsHandler  *rpc.Server  // Websocket RPC request handler to process the API requests

	stop chan struct{} // Channel to wait for termination notifications
	lock sync.RWMutex
}

func (n *Node) Accman() *accounts.Manager {
	return n.accman
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}
	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

func (n *Node) Start() error {
	log.Info("Start P2P server...")

	n.serverConfig = n.config.P2P
	n.serverConfig.LocalId = n.config.Netid
	n.serverConfig.Name = n.config.NodeName()

	privateKey := n.config.NodeKey()
	n.serverConfig.LocalId = n.config.PubkeyID(&privateKey.PublicKey)

	if n.serverConfig.NodeDatabase == "" {
		n.serverConfig.NodeDatabase = n.config.NodeDB()
	}

	running := &p2p.Server{Config: n.serverConfig}

	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctx := &ServiceContext{
			config:   n.config,
			services: make(map[reflect.Type]Service),
			EventMux: n.eventmux,
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.services[kind] = s
		}
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return &DuplicateServiceError{Kind: kind}
		}
		services[kind] = service
	}
	// Gather the protocols and start the freshly assembled P2P server
	for _, service := range services {
		running.Protocols = append(running.Protocols, service.Protocols()...)
	}

	if err := running.Start(); err != nil {

		log.Warn("p2p  Start faieled! ", "err", err)
		return err
	}

	// Start each of the services
	started := []reflect.Type{}
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.Start(running); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}
			running.Stop()

			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}

	//if err := n.startRPC(services); err != nil {
	//	for _, service := range services {
	//		service.Stop()
	//	}
	//	running.Stop()
	//	return err
	//}

	// Finish initializing the startup
	n.services = services
	n.server = running
	n.stop = make(chan struct{})

	return nil
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (n *Node) Wait() {
	n.lock.RLock()
	if n.server == nil {
		n.lock.RUnlock()
		return
	}
	stop := n.stop
	n.lock.RUnlock()

	<-stop
}

func (n *Node) Stop() error {
	log.Info("Stop P2P server...")

	// Terminate the API, services and the p2p server.
	//n.stopWS()
	//n.stopHTTP()
	//n.stopIPC()
	//n.rpcAPIs = nil

	n.server.Stop()
	n.server = nil

	return nil
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server, life cycle management
// should be left to this Node entity.
func (n *Node) Server() *p2p.Server {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.server
}

func New(conf *Config) (*Node, error) {
	am, ephemeralKeystore, _ := makeAccountManager(conf)
	return &Node{
		accman:            am,
		ephemeralKeystore: ephemeralKeystore,
		config:            conf,
		ipcEndpoint:       conf.IPCEndpoint(),
		httpEndpoint:      conf.HTTPEndpoint(),
		wsEndpoint:        conf.WSEndpoint(),
	}, nil
}

func (n *Node) DataDir() string {
	return n.config.DataDir
}

func (n *Node) IPCEndpoint() string {
	return n.ipcEndpoint
}

func (n *Node) startRPC(services map[reflect.Type]Service) error {

	var apis []rpc.API
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	// Start the various API endpoints, terminating all in case of errors
	if err := n.startInProc(apis); err != nil {
		return err
	}
	if err := n.startIPC(apis); err != nil {
		n.stopInProc()
		return err
	}
	if err := n.startHTTP(n.httpEndpoint, apis, n.config.HTTPModules, n.config.HTTPCors, n.config.HTTPVirtualHosts); err != nil {
		n.stopIPC()
		n.stopInProc()
		return err
	}
	if err := n.startWS(n.wsEndpoint, apis, n.config.WSModules, n.config.WSOrigins, n.config.WSExposeAll); err != nil {
		n.stopHTTP()
		n.stopIPC()
		n.stopInProc()
		return err
	}

	// All API endpoints started successfully
	n.rpcAPIs = apis
	return nil
}

// startInProc initializes an in-process RPC endpoint.
func (n *Node) startInProc(apis []rpc.API) error {
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		//n.log.Debug("InProc registered", "service", api.Service, "namespace", api.Namespace)
	}
	n.inprocHandler = handler
	return nil
}

// stopInProc terminates the in-process RPC endpoint.
func (n *Node) stopInProc() {
	if n.inprocHandler != nil {
		n.inprocHandler.Stop()
		n.inprocHandler = nil
	}
}

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startIPC(apis []rpc.API) error {
	if n.ipcEndpoint == "" {
		return nil // IPC disabled.
	}
	listener, handler, err := rpc.StartIPCEndpoint(n.ipcEndpoint, apis)
	if err != nil {
		return err
	}
	n.ipcListener = listener
	n.ipcHandler = handler
	//n.log.Info("IPC endpoint opened", "url", n.ipcEndpoint)
	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (n *Node) stopIPC() {
	if n.ipcListener != nil {
		n.ipcListener.Close()
		n.ipcListener = nil

		//n.log.Info("IPC endpoint closed", "endpoint", n.ipcEndpoint)
	}
	if n.ipcHandler != nil {
		n.ipcHandler.Stop()
		n.ipcHandler = nil
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	fmt.Println("HTTP : ", endpoint)
	listener, handler, err := rpc.StartHTTPEndpoint(endpoint, apis, modules, cors, vhosts)
	if err != nil {
		return err
	}
	//n.log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%s", endpoint), "cors", strings.Join(cors, ","), "vhosts", strings.Join(vhosts, ","))
	fmt.Println("HTTP endpoint opened", "url", fmt.Sprintf("http://%s", endpoint), "cors", strings.Join(cors, ","), "vhosts", strings.Join(vhosts, ","))
	// All listeners booted successfully
	n.httpEndpoint = endpoint
	n.httpListener = listener
	n.httpHandler = handler

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (n *Node) stopHTTP() {
	if n.httpListener != nil {
		n.httpListener.Close()
		n.httpListener = nil

		//n.log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%s", n.httpEndpoint))
		fmt.Println("HTTP endpoint closed", "url", fmt.Sprintf("http://%s", n.httpEndpoint))
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (n *Node) startWS(endpoint string, apis []rpc.API, modules []string, wsOrigins []string, exposeAll bool) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	fmt.Println(" WebSocket : ", endpoint)
	listener, handler, err := rpc.StartWSEndpoint(endpoint, apis, modules, wsOrigins, exposeAll)
	if err != nil {
		return err
	}
	//n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", listener.Addr()))
	fmt.Println("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", listener.Addr()))
	// All listeners booted successfully
	n.wsEndpoint = endpoint
	n.wsListener = listener
	n.wsHandler = handler

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (n *Node) stopWS() {
	if n.wsListener != nil {
		n.wsListener.Close()
		n.wsListener = nil

		//n.log.Info("WebSocket endpoint closed", "url", fmt.Sprintf("ws://%s", n.wsEndpoint))
		fmt.Println("WebSocket endpoint closed", "url", fmt.Sprintf("ws://%s", n.wsEndpoint))
	}
	if n.wsHandler != nil {
		n.wsHandler.Stop()
		n.wsHandler = nil
	}
}
