package functions

import (
	"gocoin/libraries/common"
)

type CreditMSG struct {
	ID   int
	Sign [64]byte
	From common.Address
	Data []byte
}

func GetMessage(hash common.Hash) (cm *CreditMSG) {
	return cm
}
