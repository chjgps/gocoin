package peer

import (
	flogging "gocoin/libraries/log"
	"gocoin/libraries/chaincodecore/common/comm"
	)

var peerLogger = flogging.New("peer")

var peerServer comm.GRPCServer
// CreatePeerServer creates an instance of comm.GRPCServer
// This server is used for peer communications
func CreatePeerServer(listenAddress string) (comm.GRPCServer, error) {

	var err error
	peerServer, err = comm.NewGRPCServer(listenAddress)
	if err != nil {
		peerLogger.Error("Failed to create peer server (%s)", err)
		return nil, err
	}
	return peerServer, nil
}

