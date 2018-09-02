package common

import (
	"gocoin/libraries/crypto/sha3"
)

func CalMerkel(tree []Hash, proofs *[]Hash, id uint64) Hash {
	if len(tree) == 1 {
		return tree[0]
	}
	if len(tree)%2 == 1 {
		tree = append(tree, tree[len(tree)-1])
	}

	if proofs != nil {
		if id%2 == 1 {
			*proofs = append(*proofs, tree[id-1])
		} else {
			*proofs = append(*proofs, tree[id+1])
		}
	}

	hash := make([]Hash, len(tree)/2)
	k := 0
	for i := 0; i < len(tree); i += 2 {
		hw := sha3.NewKeccak256()
		hw.Write(tree[i][:])
		hw.Write(tree[i+1][:])
		hw.Sum(hash[k][:0])
		k++
	}

	return CalMerkel(hash[:], proofs, id/2)
}

func VerigyMerkel(root Hash, proofs []Hash, hash Hash, idx uint64) bool {
	for _, h := range proofs {

		hw := sha3.NewKeccak256()
		if idx%2 == 0 {
			hw.Write(hash[:])
			hw.Write(h[:])
			hw.Sum(hash[:0])
			idx = idx / 2
		} else {
			hw.Write(h[:])
			hw.Write(hash[:])
			hw.Sum(hash[:0])
			idx = idx / 2
		}
	}

	return hash == root
}

func MerkekRoot(tree []Hash) Hash {
	return CalMerkel(tree[:], nil, 0)
}

func Root(tree []Hash) Hash {
	if len(tree) == 1 {
		return tree[0]
	}
	if len(tree)%2 == 1 {
		tree = append(tree, tree[len(tree)-1])
	}
	result := make([]Hash, len(tree)/2)

	for {
		k := 0
		for i := 0; i < len(tree); i += 2 {
			hw := sha3.NewKeccak256()
			hw.Write(tree[i][:])
			hw.Write(tree[i+1][:])
			hw.Sum(result[k][:0])
			k++
		}
		if len(result) == 1 {
			return result[0]
		}
		tree = tree[:0]
		tree = append(tree, result...)
		if len(result)%2 == 1 {
			tree = append(tree, result[len(result)-1])
		}
		result = result[:len(tree)/2]
	}
}
