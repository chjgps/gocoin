package comm

import (
	    "net"
	    "errors"
	     "google.golang.org/grpc"
	     "sync"
)

type grpcServerImpl struct {
	//Listen address for the server specified as hostname:port
	address string
	//Listener for handling network requests
	listener net.Listener
	//GRPC server
	server *grpc.Server

	lock *sync.Mutex
}

//GRPCServer defines an interface representing a GRPC-based server
type GRPCServer interface {
	//Address returns the listen address for the GRPCServer
	Address() string
	//Start starts the underlying grpc.Server
	Start() error
	//Stop stops the underlying grpc.Server
	Stop()
	//Server returns the grpc.Server instance for the GRPCServer
	Server() *grpc.Server
	//Listener returns the net.Listener instance for the GRPCServer
	Listener() net.Listener

}


//NewGRPCServer creates a new implementation of a GRPCServer given a
//listen address.
func NewGRPCServer(address string) (GRPCServer, error) {

	if address == "" {
		return nil, errors.New("Missing address parameter")
	}
	//create our listener
	lis, err := net.Listen("tcp", address)

	if err != nil {
		return nil, err
	}

	return NewGRPCServerFromListener(lis)

}

//NewGRPCServerFromListener creates a new implementation of a GRPCServer given
//an existing net.Listener instance.
func NewGRPCServerFromListener(listener net.Listener) (GRPCServer, error) {

	grpcServer := &grpcServerImpl{
		address:  listener.Addr().String(),
		listener: listener,
		lock:     &sync.Mutex{},
	}

	//set up our server options
	var serverOpts []grpc.ServerOption

	// set max send and recv msg sizes
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(MaxSendMsgSize()))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(MaxRecvMsgSize()))
	// set the keepalive options
	serverOpts = append(serverOpts, ServerKeepaliveOptions()...)

	grpcServer.server = grpc.NewServer(serverOpts...)

	return grpcServer, nil
}

//Address returns the listen address for this GRPCServer instance
func (gServer *grpcServerImpl) Address() string {
	return gServer.address
}

//Listener returns the net.Listener for the GRPCServer instance
func (gServer *grpcServerImpl) Listener() net.Listener {
	return gServer.listener
}

//Server returns the grpc.Server for the GRPCServer instance
func (gServer *grpcServerImpl) Server() *grpc.Server {
	return gServer.server
}

//Start starts the underlying grpc.Server
func (gServer *grpcServerImpl) Start() error {
	return gServer.server.Serve(gServer.listener)
}

//Stop stops the underlying grpc.Server
func (gServer *grpcServerImpl) Stop() {
	gServer.server.Stop()
}
