package comm

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"time"
)
const defaultTimeout = time.Second * 3000

// NewClientConnectionWithAddress Returns a new grpc.ClientConn to the given address.
func NewClientConnectionWithAddress(peerAddress string, block bool, tslEnabled bool, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())

	opts = append(opts, grpc.WithTimeout(defaultTimeout))
	if block {
		opts = append(opts, grpc.WithBlock())
	}
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxRecvMsgSize()),
		grpc.MaxCallSendMsgSize(MaxSendMsgSize())))
	conn, err := grpc.Dial(peerAddress, opts...)
	if err != nil {
		return nil, err
	}
	return conn, err
}

