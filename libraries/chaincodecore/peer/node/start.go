package node

import (
	flogging "gocoin/libraries/log"
	"github.com/spf13/viper"
	"gocoin/libraries/chaincodecore/peer"
	"time"
	"google.golang.org/grpc"
	"gocoin/libraries/chaincodecore/chaincode"
	pb "gocoin/libraries/chaincodecore/protos/peer"
)


var logger = flogging.New("nodeCmd")
func Start(args []string) error{

	viper.Set("peer.Address", "127.0.0.1:6052")
	viper.Set("peer.chaincodeListenAddress", "127.0.0.1:6052")
	viper.Set("peer.fileSystemPath", "/tmp/hyperledger/test")
	viper.Set("chaincode.executetimeout", "30s")
	viper.Set("chaincode.id.name", "test")
	listenAddr := viper.GetString("peer.chaincodeListenAddress")
	peerServer, err := peer.CreatePeerServer(listenAddr)
	if err != nil {
		logger.Crit("Failed to create peer server (%s)", err)
	}

	registerChaincodeSupport(peerServer.Server())
	go peerServer.Start()

	logger.Debug("Running peer")
	return err
}

//NOTE - when we implment JOIN we will no longer pass the chainID as param
//The chaincode support will come up without registering system chaincodes
//which will be registered only during join phase.
func registerChaincodeSupport(grpcServer *grpc.Server) {
	//get user mode
	//userRunsCC := chaincode.IsDevMode()

	//get chaincode startup timeout
	ccStartupTimeout := viper.GetDuration("chaincode.startuptimeout")
	if ccStartupTimeout < time.Duration(5)*time.Second {
		logger.Warn("Invalid chaincode startup timeout value %s (should be at least 5s); defaulting to 5s", ccStartupTimeout)
		ccStartupTimeout = time.Duration(5) * time.Second
	} else {
		logger.Debug("Chaincode startup timeout value set to %s", ccStartupTimeout)
	}

	ccSrv := chaincode.NewChaincodeSupport( ccStartupTimeout)


	pb.RegisterChaincodeSupportServer(grpcServer, ccSrv)
}


