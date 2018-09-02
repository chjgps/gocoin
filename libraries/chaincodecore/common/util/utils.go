package util

import (
	"gocoin/libraries/crypto/sha3"
)

func ComputeSHA256(data []byte) (hash []byte) {
 	b32 := sha3.Sum256(data)
 	hash = b32[:]
 	return hash


}

// ToChaincodeArgs converts string args to []byte args
func ToChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}