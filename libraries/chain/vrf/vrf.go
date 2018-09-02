// VRF.go
package vrf

import (
	"fmt"
	"gocoin/libraries/common"
	"gocoin/libraries/crypto/sha3"
	"gocoin/libraries/rlp"
	"math/big"
)

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

//对地址排序的一种方法
func sortByHex(addresses []common.Address, l int, r int) {
	if l < r {
		pivot := addresses[(l+r)/2].Hex()
		i := l
		j := r
		var tmp common.Address
		for i <= j {
			for addresses[i].Hex() < pivot {
				i++
			}
			for addresses[j].Hex() > pivot {
				j--
			}
			if i <= j {
				tmp = addresses[i]
				addresses[i] = addresses[j]
				addresses[j] = tmp
				i++
				j--
			}
		}
		if l < j {
			sortByHex(addresses, l, j)
		}
		if i < r {
			sortByHex(addresses, i, r)
		}
	}
}

// SortAddresses - Sort a list of address.
func SortAddresses(addresses []common.Address) {
	n := len(addresses)
	sortByHex(addresses, 0, n-1)
}

type Vrf struct {
	GroupID uint64
	Address common.Address

	GroupAddress     []common.Address
	CoinValidateInfo map[uint64]common.Address
}

func NewVrf(id uint64, addr common.Address, groupaddrs []common.Address) *Vrf {
	vrf := &Vrf{
		GroupID:          id,
		Address:          addr,
		GroupAddress:     make([]common.Address, 0, len(groupaddrs)),
		CoinValidateInfo: make(map[uint64]common.Address),
	}

	vrf.GroupAddress = append(vrf.GroupAddress, groupaddrs...)
	SortAddresses(vrf.GroupAddress)
	return vrf
}

func (f *Vrf) GetCoinInfo() {
	for k, v := range f.CoinValidateInfo {
		fmt.Println("cid : ", k, " Address : ", v)
	}
}

func (f *Vrf) CheckValidateCoin(cids []uint64, addr common.Address) bool {

	if len(cids) < 1 {
		return false
	}

	for _, cid := range cids {
		v, flag := f.CoinValidateInfo[cid]
		if flag {
			if v != addr {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func (f *Vrf) ValidateBlock(number uint64, blockhash common.Hash, address common.Address) (float32, int) {

	//fmt.Println("VRf - number : ",number,"VRf - blockhash : ",blockhash)
	//fmt.Println("Vrf - GroupAddress : ",f.GroupAddress)

	hash := rlpHash([]interface{}{
		number,
		blockhash,
	})

	b := hash.Big()
	n := len(f.GroupAddress)
	b.Mod(b, big.NewInt(int64(n)))
	index := int(b.Int64())

	//fmt.Println("VRf - number : ",number,"VRf - blockhash : ",blockhash,"VRf - index : ",index)

	var (
		coefficient float32 = 1.0
		sortIndex   int     = 0
		sum         int     = 1
	)
	for i, addr := range f.GroupAddress {
		if addr == address {
			if i < index {
				sortIndex = len(f.GroupAddress) - index + i
			} else {
				sortIndex = i - index
			}
		}
	}

	if sortIndex > 10 {
		sortIndex = 10
	}
	for k := 0; k < sortIndex; k++ {
		sum = sum << 1
	}
	coefficient = coefficient / float32(sum)

	return coefficient, sortIndex
}

func (f *Vrf) ValidateCoin(cids []uint64, txhash, blockhash common.Hash) []uint64 {

	validateCid := make([]uint64, 0, len(cids))

	for k, _ := range f.CoinValidateInfo {
		delete(f.CoinValidateInfo, k)
	}

	var (
		continuity int    = 0
		oldcid     uint64 = 0
		hash       common.Hash
		addFlag    bool = false
		addr       common.Address
	)

	for _, cid := range cids {

		if oldcid > 0 {
			if oldcid+1 == cid {
				continuity = continuity + 1
				if continuity > 4 {
					continuity = 0
				} else {

					oldcid = cid
					if addFlag {
						validateCid = append(validateCid, cid)
					}

					f.CoinValidateInfo[cid] = addr
					continue
				}
			}
		}

		hash = rlpHash([]interface{}{
			cid,
			txhash,
			blockhash,
		})

		oldcid = cid
		addFlag = false
		addr = f.validateAddress(hash)
		f.CoinValidateInfo[cid] = addr
		if addr == f.Address {
			addFlag = true
			validateCid = append(validateCid, cid)
		}
	}

	return validateCid
}

func (f *Vrf) MultipleValidateCoin(times int, cids []uint64, txhash, blockhash common.Hash) []uint64 {

	validateCid := make([]uint64, 0, len(cids))
	var (
		continuity int    = 0
		oldcid     uint64 = 0
		hash       common.Hash
		addFlag    bool = false
	)

	for _, cid := range cids {

		if oldcid > 0 {

			if oldcid+1 == cid {

				continuity = continuity + 1
				if continuity > 5 {
					continuity = 0
				} else {

					oldcid = cid
					if addFlag {
						validateCid = append(validateCid, cid)
					}
					continue
				}
			}
		}

		hash = rlpHash([]interface{}{
			cid,
			txhash,
			blockhash,
		})

		oldcid = cid
		addFlag = false
		addr := f.multiplevalidateAddress(times, hash)
		if addr == f.Address {
			addFlag = true
			validateCid = append(validateCid, cid)
		}
	}

	return validateCid
}

func (f *Vrf) multiplevalidateAddress(times int, hash common.Hash) common.Address {

	b := hash.Big()
	n := len(f.GroupAddress)
	b.Mod(b, big.NewInt(int64(n)))
	index := int(b.Int64()) + times

	if index >= n {
		index = index - n
	}

	return f.GroupAddress[index]
}

func (f *Vrf) validateAddress(hash common.Hash) common.Address {

	b := hash.Big()
	n := len(f.GroupAddress)
	b.Mod(b, big.NewInt(int64(n)))
	index := int(b.Int64())
	return f.GroupAddress[index]
}

func (f *Vrf) GetGroupID() uint64                { return f.GroupID }
func (f *Vrf) GetAddress() common.Address        { return f.Address }
func (f *Vrf) GetGroupAddress() []common.Address { return f.GroupAddress }
