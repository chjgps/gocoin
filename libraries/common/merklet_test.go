package common

import (
	"gocoin/libraries/crypto/sha3"
	"testing"
)

func Test_merkle(t *testing.T) {
	h1 := StringToHash("1")
	h2 := StringToHash("2")
	h3 := StringToHash("3")
	h4 := StringToHash("4")
	h5 := StringToHash("5")

	hash := MerkekRoot([]Hash{h1, h2, h3, h4, h5})
	t.Log(hash)

	hashRoot := Root([]Hash{h1, h2, h3, h4, h5})
	t.Log(hash)

	var h12 Hash
	hw := sha3.NewKeccak256()
	hw.Write(h1[:])
	hw.Write(h2[:])
	hw.Sum(h12[:0])

	var h34 Hash
	hw = sha3.NewKeccak256()
	hw.Write(h3[:])
	hw.Write(h4[:])
	hw.Sum(h34[:0])

	var h55 Hash
	hw = sha3.NewKeccak256()
	hw.Write(h5[:])
	hw.Write(h5[:])
	hw.Sum(h55[:0])

	var h1234 Hash
	hw = sha3.NewKeccak256()
	hw.Write(h12[:])
	hw.Write(h34[:])
	hw.Sum(h1234[:0])

	var h5555 Hash
	hw = sha3.NewKeccak256()
	hw.Write(h55[:])
	hw.Write(h55[:])
	hw.Sum(h5555[:0])

	var root Hash
	hw = sha3.NewKeccak256()
	hw.Write(h1234[:])
	hw.Write(h5555[:])
	hw.Sum(root[:0])
	t.Log(root)

	if root != hash {
		t.Error(root, hash)
	}

	if root != hashRoot {
		t.Error(root, hashRoot)
	}
}
