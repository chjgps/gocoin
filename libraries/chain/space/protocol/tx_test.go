/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.12.3
 * Website    : http:www.gocoin.com
 * Function   : transaction  Test
**/

package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"testing"

	"gocoin/libraries/common"
	"gocoin/libraries/crypto"
	"gocoin/libraries/rlp"
)

var (
	emptyTx = NewTx(
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		big.NewInt(0), big.NewInt(0),
		nil,
	)

	rightvrsTx, _ = NewTx(
		common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
		big.NewInt(2000),
		big.NewInt(10),
		common.FromHex("5544"),
	).WithSignature(
		FrontierSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)
)

func TestTransactionSigHash(t *testing.T) {
	var frontier FrontierSigner
	if frontier.Hash(emptyTx) != common.HexToHash("e5c1425b12bc2c451b75f275b4e53ca9879caa89665555d5700e687b88212ccd") {
		t.Errorf("empty transaction hash mismatch, got %x", emptyTx.Hash())
	}
	if frontier.Hash(rightvrsTx) != common.HexToHash("f3dde2027b3804aaa5154c8673784e90c77f373cf3206a24c8a78ea1fb94634c") {
		t.Errorf("RightVRS transaction hash mismatch, got %x", rightvrsTx.Hash())
	}
}

func TestTransactionEncode(t *testing.T) {
	txb, err := rlp.EncodeToBytes(rightvrsTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	should := common.FromHex("f85f0a94b94f5374fce5edbc8e2a8697c15331677e6ebf0b8207d08255441ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3")
	if !bytes.Equal(txb, should) {
		t.Errorf("encoded RLP mismatch, got %x", txb)
	}
}

func decodeTx(data []byte) (*Tx, error) {
	var tx Tx
	t, err := &tx, rlp.Decode(bytes.NewReader(data), &tx)

	return t, err
}

func defaultTestKey() (*ecdsa.PrivateKey, common.Address) {
	key, _ := crypto.HexToECDSA("45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}

func TestRecipientEmpty(t *testing.T) {
	_, addr := defaultTestKey()
	tx, err := decodeTx(common.Hex2Bytes("f85b8094000000000000000000000000000000000000000080801ca0f77003641a71c880672eb6c1aade626efa2335174ee6f612d9303bf21e3c83d2a0200c10f355a7123d104e8619241440b9815bd6ccc1a88f9f4af4a794e6c94f5e"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	from, err := Sender(FrontierSigner{}, tx)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if addr != from {
		t.Error("derived address doesn't match")
	}
}

func TestRecipientNormal(t *testing.T) {
	_, addr := defaultTestKey()

	tx, err := decodeTx(common.Hex2Bytes("f85f0a94b94f5374fce5edbc8e2a8697c15331677e6ebf0b8207d08255441ba084a89d160a50600bc141c4b3dc372628eea65ba85af3ade606b906c4927d766fa02a8fa9db4c4865f65675ec54f9a6aed74fb68e649470f68153b87309575c5f54"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	from, err := Sender(FrontierSigner{}, tx)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if addr != from {
		t.Error("derived address doesn't match")
	}
}

func TestTransactionJSON(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	signer := MakeSigner()

	for i := uint64(0); i < 25; i++ {
		var tx *Tx
		switch i % 2 {
		case 0:
			tx = NewTx(common.Address{1}, common.Big0, common.Big1, []byte("abcdef"))
		case 1:
			tx = NewSpaceCreation(common.Big0, common.Address{1})
		}

		tx, err := SignTx(tx, signer, key)
		if err != nil {
			t.Fatalf("could not sign transaction: %v", err)
		}

		data, err := json.Marshal(tx)
		if err != nil {
			t.Errorf("json.Marshal failed: %v", err)
		}

		var parsedTx *Tx
		if err := json.Unmarshal(data, &parsedTx); err != nil {
			t.Errorf("json.Unmarshal failed: %v", err)
		}

		if tx.Hash() != parsedTx.Hash() {
			t.Errorf("parsed tx differs from original tx, want %v, got %v", tx, parsedTx)
		}

	}
}

func TestSpace(t *testing.T) {

	wantAddr := common.Address{1}
	txSpaceCreation := NewSpaceCreation(big.NewInt(2000), wantAddr)

	var dataSpaceCreation struct {
		T inputDataType
		A common.Address
	}

	if err := rlp.DecodeBytes(txSpaceCreation.Data(), &dataSpaceCreation); err != nil {
		t.Errorf("rlp.DecodeBytes failed: %v", err)
	}

	if dataSpaceCreation.T != idtSpaceCreation {
		t.Errorf("Type differs: want:%d, got:%d", idtSpaceCreation, dataSpaceCreation.T)
	}
	if dataSpaceCreation.A.Hash() != wantAddr.Hash() {
		t.Errorf("Addr differs: want:%v, got:%v", wantAddr.Str(), dataSpaceCreation.A.Str())
	}

	txRecharge := NewRecharge(wantAddr, big.NewInt(2000), big.NewInt(10))

	var dataRecharge struct {
		T inputDataType
		A common.Address
	}

	if err := rlp.DecodeBytes(txRecharge.Data(), &dataRecharge); err != nil {
		t.Errorf("rlp.DecodeBytes failed: %v", err)
	}

	if dataRecharge.T != idtRecharge {
		t.Errorf("Type differs: want:%d, got:%d", idtRecharge, dataRecharge.T)
	}
	if dataRecharge.A.Hash() != wantAddr.Hash() {
		t.Errorf("Addr differs: want:%v, got:%v", wantAddr.Str(), dataRecharge.A.Str())
	}

	txAttrSet := NewAttrSet(wantAddr, big.NewInt(2000), Attr{[]byte("wanglei"), []byte("test")})

	var dataAttrSet struct {
		T    inputDataType
		A    common.Address
		Attr Attr
	}

	if err := rlp.DecodeBytes(txAttrSet.Data(), &dataAttrSet); err != nil {
		t.Errorf("rlp.DecodeBytes failed: %v", err)
	}

	if dataAttrSet.T != idtAttrSet {
		t.Errorf("Type differs: want:%d, got:%d", idtAttrSet, dataSpaceCreation.T)
	}
	if dataAttrSet.A.Hash() != wantAddr.Hash() {
		t.Errorf("Addr differs: want:%v, got:%v", wantAddr.Str(), dataSpaceCreation.A.Str())
	}
	if string(dataAttrSet.Attr.K) != "wanglei" {
		t.Errorf("Addr differs: want:%v, got:%v", "wanglei", dataAttrSet.Attr.K)
	}
	if string(dataAttrSet.Attr.V) != "test" {
		t.Errorf("Addr differs: want:%v, got:%v", "test", dataAttrSet.Attr.V)
	}
}
