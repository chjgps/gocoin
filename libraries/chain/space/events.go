/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : block chain feed event
**/

package space

import (
	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/common"
)

type BlockSigEvent struct {
	Hash common.Hash
	Sig  *BlockSigInfo
}

type LastBlockEvent struct {
	Hash  common.Hash
	Block *protocol.Block
}

type NewBlockEvent struct {
	Hash common.Hash
	Head *protocol.Block
}
