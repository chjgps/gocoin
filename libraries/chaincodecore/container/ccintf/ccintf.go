package ccintf

import (
	"encoding/hex"
	pb "gocoin/libraries/chaincodecore/protos/peer"
	"gocoin/libraries/chaincodecore/common/util"
	"golang.org/x/net/context"
)


// ChaincodeStream interface for stream between Peer and chaincode instance.
type ChaincodeStream interface {
	Send(*pb.ChaincodeMessage) error
	Recv() (*pb.ChaincodeMessage, error)
}

// CCSupport must be implemented by the chaincode support side in peer
// (such as chaincode_support)
type CCSupport interface {
	HandleChaincodeStream(context.Context, ChaincodeStream) error
}


// GetCCHandlerKey is used to pass CCSupport via context
func GetCCHandlerKey() string {
	return "CCHANDLER"
}

//CCID encapsulates chaincode ID
type CCID struct {
	ChaincodeSpec *pb.ChaincodeSpec
	NetworkID     string
	PeerID        string
	ChainID       string
	Version       string
}

//GetName returns canonical chaincode name based on chain name
func (ccid *CCID) GetName() string {
	if ccid.ChaincodeSpec == nil {
		panic("nil chaincode spec")
	}

	name := ccid.ChaincodeSpec.ChaincodeId.Name
	if ccid.Version != "" {
		name = name + "-" + ccid.Version
	}

	//this better be chainless system chaincode!
	if ccid.ChainID != "" {
		hash := util.ComputeSHA256([]byte(ccid.ChainID))
		hexstr := hex.EncodeToString(hash[:])
		name = name + "-" + hexstr
	}

	return name
}

