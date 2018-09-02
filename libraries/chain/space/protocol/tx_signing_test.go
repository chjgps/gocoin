/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.12.3
 * Website    : http:www.gocoin.com
 * Function   : transaction  Sign  Test
**/

package protocol

import (
	"math/big"
	"testing"

	"gocoin/libraries/crypto"
)

func TestSigning(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	signer := MakeSigner()
	tx, err := SignTx(NewTx(addr, new(big.Int), new(big.Int), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(signer, tx)
	if err != nil {
		t.Fatal(err)
	}
	if from != addr {
		t.Errorf("exected from and address to be equal. Got %x want %x", from, addr)
	}
}
