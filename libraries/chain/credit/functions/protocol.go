package functions

import "gocoin/libraries/common"

const (
	BASECODE       = 0x30
	ERROR_SPACE_TX = BASECODE + 1
)

type ErrorSpaceTx struct {
	TxHash common.Hash
}
