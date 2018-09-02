/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.12.3
 * Website    : http:www.gocoin.com
 * Function   : Generation of hash roots
**/

package protocol

import (
	"bytes"

	"gocoin/libraries/chain/space/trie"
	"gocoin/libraries/common"
	"gocoin/libraries/rlp"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) common.Hash {
	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < list.Len(); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		trie.Update(keybuf.Bytes(), list.GetRlp(i))
	}
	return trie.Hash()
}
