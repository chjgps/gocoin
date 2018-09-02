package validategroup

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"strconv"

	"gocoin/libraries/common"
	"gocoin/libraries/crypto"
	"gocoin/libraries/crypto/sha3"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/rlp"
)

var (
	groupMembersKey     = []byte("LastGroupMembers")
	thresholdMembersKey = []byte("thresholdMembers")

	coinMembersKey          = []byte("coinMembers")
	coingroupMembersKey     = []byte("coinnewGroupMembers")
	thresholdCoinMembersKey = []byte("thresholdCoinMembers")

	transactionMembersKey = []byte("transactionMembers")
	newTxMembersKey       = []byte("newTxMembers")
	thresholdTxMembersKey = []byte("thresholdTxMembers")

	stellMembersKey = []byte("StellMembers")
)

const (
	ValidateLimit    = 10
	blockNumberLimet = 10 //Epoch = 10
)

type GroupMember struct {
	Address common.Address `json:"address"`
	Time    *big.Int       `json:"time"`
}

type GroupMembers []GroupMember

func (self GroupMembers) Len() int { return len(self) }
func (self GroupMembers) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}
func (self GroupMembers) Less(i, j int) bool {
	return self[i].Time.Cmp(self[j].Time) < 0
}

// RandLength --
const RandLength = 32

// Rand --
type Rand []byte

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// RandFromBytes --
func RandFromBytes(b []byte) (r Rand) {
	h := crypto.Keccak256Hash(b)
	r = make([]byte, RandLength)
	copy(r[:], h[:RandLength])
	return
}

// DerivedRand -- Derived Randomness hierarchically
func (r Rand) DerivedRand(idx []byte) Rand {
	// Keccak is not susceptible to length-extension-attacks, so we can use it as-is to implement an HMAC
	return RandFromBytes(crypto.Keccak256(r[:RandLength], idx))
}

// Shortcuts to the derivation function

// Ders --
// ... by string
func (r Rand) Ders(s ...string) Rand {
	ri := r
	for _, si := range s {
		ri = ri.DerivedRand([]byte(si))
	}
	return ri
}

// Deri --
// ... by int
func (r Rand) Deri(i int) Rand {
	return r.Ders(strconv.Itoa(i))
}

// Modulo --
// Convert to a random integer from the interval [0,n-1].
func (r Rand) Modulo(n int) int {
	// modulo len(groups) with big.Ints (Mod method works on pointers)
	//var b big.Int
	b := big.NewInt(0)
	b.SetBytes(r)
	b.Mod(b, big.NewInt(int64(n)))
	return int(b.Int64())
}

// Convert to a random permutation
func (r Rand) RandomPerm(n int, k int) []int {
	// modulo len(groups) with big.Ints (Mod method works on pointers)
	l := make([]int, n)
	for i := range l {
		l[i] = i
	}
	for i := 0; i < k; i++ {
		j := r.Deri(i).Modulo(n-i) + i
		l[i], l[j] = l[j], l[i]
	}
	return l[:k]
}

type ValidateGroup struct {
	groupID    int
	groupSize  int
	number     uint64
	noValidate bool

	//space  consensus
	groupmembers          GroupMembers
	newgroupmembers       GroupMembers
	thresholdgroupmembers GroupMembers
	consensusMembers      []common.Address
	consensusAddress      map[common.Address]struct{}
	membersAddress        map[common.Address]struct{}

	//transaction  coin
	coingroupmembers          GroupMembers
	newcoingroupmembers       GroupMembers
	thresholdcoingroupmembers GroupMembers

	validateMembers []common.Address
	validateAddress map[common.Address]struct{}
	coinAddresses   map[common.Address]struct{}

	//transaction consensus
	transactiongroupmembers GroupMembers
	newtxgroupmembers       GroupMembers
	thresholdtxgroupmembers GroupMembers
	consensusTxMembers      []common.Address
	consensusTxAddress      map[common.Address]struct{}
	transactionAddresses    map[common.Address]struct{}

	stellgroupmembers GroupMembers
	stellAddresses    map[common.Address]struct{}

	groupDb lvldb.Database
}

func NewValidateGroup(id int, hash common.Hash, db lvldb.Database) *ValidateGroup {
	vg := &ValidateGroup{
		groupID:              id,
		groupDb:              db,
		noValidate:           false,
		validateMembers:      make([]common.Address, 0, ValidateLimit),
		consensusMembers:     make([]common.Address, 0, ValidateLimit),
		membersAddress:       make(map[common.Address]struct{}),
		validateAddress:      make(map[common.Address]struct{}),
		consensusAddress:     make(map[common.Address]struct{}),
		stellAddresses:       make(map[common.Address]struct{}),
		coinAddresses:        make(map[common.Address]struct{}),
		transactionAddresses: make(map[common.Address]struct{}),
		consensusTxMembers:   make([]common.Address, 0, ValidateLimit),
		consensusTxAddress:   make(map[common.Address]struct{}),
	}

	//fmt.Println("GroupConst : ", hash)
	err, size, members := GetGroupConst(hash, vg.groupDb)
	vg.groupmembers = make(GroupMembers, 0, size)

	if err != nil {
		vg.noValidate = false
	} else {
		if size > 0 {
			vg.groupmembers = append(vg.groupmembers, members...)
			for _, member := range vg.groupmembers {
				vg.membersAddress[member.Address] = struct{}{}
			}
			sort.Sort(vg.groupmembers)
			tmphash := rlpHash(vg.groupmembers)
			if tmphash == hash {
				vg.noValidate = true
			}
		}
		vg.groupSize = size
	}

	err, size, members = GetThresholdGroup(vg.groupDb)
	vg.thresholdgroupmembers = make(GroupMembers, 0, size)
	if size > 0 {
		vg.thresholdgroupmembers = append(vg.thresholdgroupmembers, members...)
		for _, member := range vg.thresholdgroupmembers {
			vg.membersAddress[member.Address] = struct{}{}
		}
	}

	err, size, members = GetValidateGroup(vg.groupDb)
	vg.newgroupmembers = make(GroupMembers, 0, size)
	if size > 0 {

		vg.newgroupmembers = append(vg.newgroupmembers, members...)
		for _, member := range vg.newgroupmembers {
			vg.membersAddress[member.Address] = struct{}{}
		}
	}

	err, size, members = GetStellGroup(vg.groupDb)
	vg.stellgroupmembers = make(GroupMembers, 0, size)
	if size > 0 {

		vg.stellgroupmembers = append(vg.stellgroupmembers, members...)
		for _, member := range vg.stellgroupmembers {
			vg.stellAddresses[member.Address] = struct{}{}
		}
	}

	err, size, members = GetCoinGroup(vg.groupDb)
	vg.coingroupmembers = make(GroupMembers, 0, size)
	if size > 0 {

		vg.coingroupmembers = append(vg.coingroupmembers, members...)
		for _, member := range vg.coingroupmembers {
			vg.coinAddresses[member.Address] = struct{}{}
		}
	}

	err, size, members = GetCoinValidateGroup(vg.groupDb)
	vg.newcoingroupmembers = make(GroupMembers, 0, size)
	if size > 0 {

		vg.newcoingroupmembers = append(vg.newcoingroupmembers, members...)
		for _, member := range vg.newcoingroupmembers {
			vg.coinAddresses[member.Address] = struct{}{}
		}
	}

	err, size, members = GetThresholdCoinGroup(vg.groupDb)
	vg.thresholdcoingroupmembers = make(GroupMembers, 0, size)
	if size > 0 {
		vg.thresholdcoingroupmembers = append(vg.thresholdcoingroupmembers, members...)
		for _, member := range vg.thresholdcoingroupmembers {
			vg.coinAddresses[member.Address] = struct{}{}
		}
	}

	err, size, members = GetTransactionGroup(vg.groupDb)
	vg.transactiongroupmembers = make(GroupMembers, 0, size)
	if size > 0 {

		vg.transactiongroupmembers = append(vg.transactiongroupmembers, members...)
		for _, member := range vg.transactiongroupmembers {
			vg.transactionAddresses[member.Address] = struct{}{}
		}
	}

	err, size, members = GetNewTxMembers(vg.groupDb)
	vg.newtxgroupmembers = make(GroupMembers, 0, size)
	if size > 0 {

		vg.newtxgroupmembers = append(vg.newtxgroupmembers, members...)
		for _, member := range vg.newtxgroupmembers {
			vg.transactionAddresses[member.Address] = struct{}{}
		}
	}

	err, size, members = GetThresholdTxMembers(vg.groupDb)
	vg.thresholdtxgroupmembers = make(GroupMembers, 0, size)
	if size > 0 {
		vg.thresholdtxgroupmembers = append(vg.thresholdtxgroupmembers, members...)
		for _, member := range vg.thresholdtxgroupmembers {
			vg.transactionAddresses[member.Address] = struct{}{}
		}
	}

	return vg
}

func (vg *ValidateGroup) Add(member GroupMember) {

	_, ok := vg.membersAddress[member.Address]
	fmt.Println("add GroupMember state : ", ok)

	if !ok {

		vg.membersAddress[member.Address] = struct{}{}
		if len(vg.newgroupmembers) > 0 {

			last := vg.newgroupmembers[len(vg.newgroupmembers)-1]

			if last.Time.Cmp(member.Time) < 0 {
				vg.newgroupmembers = append(vg.newgroupmembers, member)
				WriteValidateGroup(vg.groupDb, vg.newgroupmembers)
			}
		} else {

			vg.newgroupmembers = append(vg.newgroupmembers, member)
			WriteValidateGroup(vg.groupDb, vg.newgroupmembers)
		}
	}
}

func (vg *ValidateGroup) AddStell(member GroupMember) {

	_, ok := vg.stellAddresses[member.Address]
	fmt.Println("add  stellGroupMember  state : ", ok)

	if !ok {

		vg.stellAddresses[member.Address] = struct{}{}
		if len(vg.stellgroupmembers) > 0 {

			last := vg.stellgroupmembers[len(vg.stellgroupmembers)-1]

			if last.Time.Cmp(member.Time) < 0 {
				vg.stellgroupmembers = append(vg.stellgroupmembers, member)
				WriteStellGroup(vg.groupDb, vg.stellgroupmembers)
			}
		} else {

			vg.stellgroupmembers = append(vg.stellgroupmembers, member)
			WriteStellGroup(vg.groupDb, vg.stellgroupmembers)
		}
	}
}

func (vg *ValidateGroup) IsStellAddress(addr common.Address) bool {

	if len(vg.stellAddresses) > 0 {
		_, ok := vg.stellAddresses[addr]
		return ok
	}
	return false
}

func (vg *ValidateGroup) GetStellGroupSize() uint64 {
	return uint64(len(vg.stellgroupmembers))
}

func (vg *ValidateGroup) GetStellGroup() GroupMembers {

	if len(vg.stellgroupmembers) > 0 {
		stellmembers := make(GroupMembers, len(vg.stellgroupmembers))
		copy(stellmembers, vg.stellgroupmembers)
		return stellmembers
	}

	stellmembers := make(GroupMembers, 0, 1)
	return stellmembers
}

func (vg *ValidateGroup) AddCoin(member GroupMember) {

	_, ok := vg.coinAddresses[member.Address]
	fmt.Println("add  coinGroupMember  state : ", ok)
	if !ok {

		vg.coinAddresses[member.Address] = struct{}{}
		if len(vg.newcoingroupmembers) > 0 {

			last := vg.newcoingroupmembers[len(vg.newcoingroupmembers)-1]

			if last.Time.Cmp(member.Time) < 0 {
				vg.newcoingroupmembers = append(vg.newcoingroupmembers, member)
				WriteCoinValidateGroup(vg.groupDb, vg.newcoingroupmembers)
			}
		} else {

			vg.newcoingroupmembers = append(vg.newcoingroupmembers, member)
			WriteCoinValidateGroup(vg.groupDb, vg.newcoingroupmembers)
		}
	}
}

func (vg *ValidateGroup) AddTransaction(member GroupMember) {

	_, ok := vg.transactionAddresses[member.Address]
	fmt.Println("add  transaction GroupMember  state : ", ok)

	if !ok {

		vg.transactionAddresses[member.Address] = struct{}{}
		if len(vg.newtxgroupmembers) > 0 {

			last := vg.newtxgroupmembers[len(vg.newtxgroupmembers)-1]

			if last.Time.Cmp(member.Time) < 0 {
				vg.newtxgroupmembers = append(vg.newtxgroupmembers, member)
				WriteNewTxMembers(vg.groupDb, vg.newtxgroupmembers)
			}
		} else {

			vg.newtxgroupmembers = append(vg.newtxgroupmembers, member)
			WriteNewTxMembers(vg.groupDb, vg.newtxgroupmembers)
		}
	}
}

func (vg *ValidateGroup) Adds(members GroupMembers) {
}

func (vg *ValidateGroup) Hash() common.Hash {
	return rlpHash(vg.groupmembers)
}
func (vg *ValidateGroup) Reset(hash common.Hash, init bool) error {

	if init {
		if len(vg.newgroupmembers) > 0 {

			vg.groupmembers = append(vg.groupmembers, vg.newgroupmembers...)
			DeleteValidateGroup(vg.groupDb)
			vg.newgroupmembers = vg.newgroupmembers[0:0]

			sort.Sort(vg.groupmembers)
			tmphash := rlpHash(vg.groupmembers[0].Address)
			fmt.Println("groupmembers : ", vg.groupmembers)

			if tmphash == hash {
				vg.noValidate = true
				fmt.Println("block GroupConstHash : ", hash)
				WriteGroupConst(vg.groupDb, hash, vg.groupmembers)
			} else {
				vg.noValidate = false
			}
		}

		vg.consensusMembers = vg.consensusMembers[0:0]
		vg.consensusAddress = make(map[common.Address]struct{})

	} else {

		var updateValidate bool = false
		if len(vg.newgroupmembers) > 0 {

			if len(vg.thresholdgroupmembers) >= 3 {
				updateValidate = true
				vg.groupmembers = make(GroupMembers, len(vg.thresholdgroupmembers))
				copy(vg.groupmembers, vg.thresholdgroupmembers)

				vg.thresholdgroupmembers = append(vg.thresholdgroupmembers, vg.newgroupmembers...)
			} else {
				vg.thresholdgroupmembers = append(vg.thresholdgroupmembers, vg.newgroupmembers...)
			}

			WriteThresholdGroup(vg.groupDb, vg.thresholdgroupmembers)
			DeleteValidateGroup(vg.groupDb)
			vg.newgroupmembers = vg.newgroupmembers[0:0]
		} else {
			if len(vg.thresholdgroupmembers) >= 3 {
				updateValidate = true
				vg.groupmembers = make(GroupMembers, len(vg.thresholdgroupmembers))
				copy(vg.groupmembers, vg.thresholdgroupmembers)
			}
		}

		sort.Sort(vg.groupmembers)
		tmphash := rlpHash(vg.groupmembers)
		fmt.Println("groupmembers : ", vg.groupmembers)
		fmt.Println("block GroupConstHash : ", hash, " hash : ", tmphash)
		vg.noValidate = true
		if updateValidate {

			if tmphash == hash {
				WriteGroupConst(vg.groupDb, hash, vg.groupmembers)
			} else {
				WriteGroupConst(vg.groupDb, hash, vg.groupmembers)
				WriteGroupConst(vg.groupDb, tmphash, vg.groupmembers)
			}

		}

		vg.consensusMembers = vg.consensusMembers[0:0]
		vg.consensusAddress = make(map[common.Address]struct{})
	}

	return nil
}

func (vg *ValidateGroup) ResetCoinGroup() error {

	var updateValidate bool = false
	fmt.Println("newcoingroupmembers : ", len(vg.newcoingroupmembers))
	if len(vg.newcoingroupmembers) > 0 {

		if len(vg.thresholdcoingroupmembers) >= 3 {
			updateValidate = true
			vg.coingroupmembers = make(GroupMembers, len(vg.thresholdcoingroupmembers))
			copy(vg.coingroupmembers, vg.thresholdcoingroupmembers)

			vg.thresholdcoingroupmembers = append(vg.thresholdcoingroupmembers, vg.newcoingroupmembers...)
		} else {
			vg.thresholdcoingroupmembers = append(vg.thresholdcoingroupmembers, vg.newcoingroupmembers...)
		}

		WriteThresholdCoinGroup(vg.groupDb, vg.thresholdcoingroupmembers)
		DeleteCoinValidateGroup(vg.groupDb)
		vg.newcoingroupmembers = vg.newcoingroupmembers[0:0]

	} else {

		if len(vg.thresholdcoingroupmembers) >= 3 {
			updateValidate = true
			vg.coingroupmembers = make(GroupMembers, len(vg.thresholdcoingroupmembers))
			copy(vg.coingroupmembers, vg.thresholdcoingroupmembers)
		}
	}

	sort.Sort(vg.coingroupmembers)
	vg.noValidate = true
	if updateValidate {
		WriteCoinGroup(vg.groupDb, vg.coingroupmembers)
	}

	vg.validateMembers = vg.validateMembers[0:0]
	vg.validateAddress = make(map[common.Address]struct{})

	return nil
}

func (vg *ValidateGroup) ResetTxGroup() error {

	var updateValidate bool = false
	fmt.Println("newtxgroupmembers : ", len(vg.newtxgroupmembers))
	if len(vg.newtxgroupmembers) > 0 {

		if len(vg.thresholdtxgroupmembers) >= 3 {
			updateValidate = true
			vg.transactiongroupmembers = make(GroupMembers, len(vg.thresholdtxgroupmembers))
			copy(vg.transactiongroupmembers, vg.thresholdtxgroupmembers)

			vg.thresholdtxgroupmembers = append(vg.thresholdtxgroupmembers, vg.newtxgroupmembers...)
		} else {
			vg.thresholdtxgroupmembers = append(vg.thresholdtxgroupmembers, vg.newtxgroupmembers...)
		}

		WriteThresholdTxMembers(vg.groupDb, vg.thresholdtxgroupmembers)
		DeleteNewTxMembers(vg.groupDb)
		vg.newtxgroupmembers = vg.newtxgroupmembers[0:0]

	} else {

		if len(vg.thresholdtxgroupmembers) >= 3 {
			updateValidate = true
			vg.transactiongroupmembers = make(GroupMembers, len(vg.thresholdtxgroupmembers))
			copy(vg.transactiongroupmembers, vg.thresholdtxgroupmembers)
		}
	}

	sort.Sort(vg.transactiongroupmembers)
	vg.noValidate = true
	if updateValidate {
		WriteTransactionGroup(vg.groupDb, vg.transactiongroupmembers)
	}

	vg.consensusTxMembers = vg.consensusTxMembers[0:0]
	vg.consensusTxAddress = make(map[common.Address]struct{})

	return nil
}

func (vg *ValidateGroup) GetThresholdSize() int {
	return len(vg.thresholdgroupmembers)
}

func (vg *ValidateGroup) GetThresholdCoinSize() int {
	return len(vg.thresholdcoingroupmembers)
}

func (vg *ValidateGroup) GetThresholdTxSize() int {
	return len(vg.thresholdtxgroupmembers)
}

func (vg *ValidateGroup) SetlastGroupHash(hash common.Hash) {

	fmt.Println("block GroupConstHash : ", hash)
	WriteGroupConst(vg.groupDb, hash, vg.groupmembers)
}

func (vg *ValidateGroup) SelectThresholds(number uint64) []common.Address {

	var thresholdMembers []common.Address
	if len(vg.thresholdgroupmembers) > 0 && len(vg.thresholdgroupmembers) < ValidateLimit {

		for _, member := range vg.thresholdgroupmembers {
			thresholdMembers = append(thresholdMembers, member.Address)
		}

	} else {

		epochNumber := uint64(number/blockNumberLimet + 1)

		r := RandFromBytes(encodeBlockNumber(epochNumber))
		size := ValidateLimit
		members := r.RandomPerm(len(vg.thresholdgroupmembers), size)

		for _, index := range members {
			thresholdMembers = append(thresholdMembers, vg.thresholdgroupmembers[index].Address)
		}

	}

	return thresholdMembers
}

func (vg *ValidateGroup) SelectConsensus(number uint64) []common.Address {

	epochNumber := uint64(number / blockNumberLimet)
	if vg.number == epochNumber {
		if len(vg.consensusMembers) > 0 {
			return vg.consensusMembers
		}
	} else {
		vg.consensusMembers = vg.consensusMembers[0:0]
	}

	if len(vg.consensusAddress) > 0 {
		for k, _ := range vg.consensusAddress {
			delete(vg.consensusAddress, k)
		}
	}

	fmt.Println("vg.consensusMembers(space) :", len(vg.groupmembers))

	if len(vg.groupmembers) > 0 {

		if len(vg.groupmembers) > 0 && len(vg.groupmembers) < ValidateLimit {

			for _, member := range vg.groupmembers {
				vg.consensusAddress[member.Address] = struct{}{}
				vg.consensusMembers = append(vg.consensusMembers, member.Address)
			}

		} else {

			r := RandFromBytes(encodeBlockNumber(epochNumber))
			size := ValidateLimit
			members := r.RandomPerm(len(vg.groupmembers), size)

			for _, index := range members {
				vg.consensusMembers = append(vg.consensusMembers, vg.groupmembers[index].Address)
				vg.consensusAddress[vg.groupmembers[index].Address] = struct{}{}
			}

		}
	}

	return vg.consensusMembers
}

func (vg *ValidateGroup) SelectCoinThresholds(number uint64) []common.Address {

	var thresholdMembers []common.Address
	if len(vg.thresholdcoingroupmembers) > 0 && len(vg.thresholdcoingroupmembers) < ValidateLimit {

		for _, member := range vg.thresholdcoingroupmembers {
			thresholdMembers = append(thresholdMembers, member.Address)
		}

	} else {

		epochNumber := uint64(number/blockNumberLimet + 1)

		r := RandFromBytes(encodeBlockNumber(epochNumber))
		size := ValidateLimit
		members := r.RandomPerm(len(vg.thresholdcoingroupmembers), size)

		for _, index := range members {
			thresholdMembers = append(thresholdMembers, vg.thresholdcoingroupmembers[index].Address)
		}

	}

	return thresholdMembers
}

func (vg *ValidateGroup) SelectValidate(number uint64) []common.Address {

	epochNumber := uint64(number / blockNumberLimet)
	if vg.number == epochNumber {
		if len(vg.validateMembers) > 0 {
			return vg.validateMembers
		}
	} else {
		vg.validateMembers = vg.validateMembers[0:0]
	}

	if len(vg.validateAddress) > 0 {
		for k, _ := range vg.validateAddress {
			delete(vg.validateAddress, k)
		}
	}

	fmt.Println("vg.groupmembers(Tx coin) :", len(vg.coingroupmembers))

	if len(vg.coingroupmembers) > 0 {
		if len(vg.coingroupmembers) > 0 && len(vg.coingroupmembers) < ValidateLimit {

			for _, member := range vg.coingroupmembers {
				vg.validateAddress[member.Address] = struct{}{}
				vg.validateMembers = append(vg.validateMembers, member.Address)
			}

		} else {

			r := RandFromBytes(encodeBlockNumber(epochNumber))
			size := ValidateLimit
			members := r.RandomPerm(len(vg.coingroupmembers), size)

			for _, index := range members {
				vg.validateMembers = append(vg.validateMembers, vg.coingroupmembers[index].Address)
				vg.validateAddress[vg.coingroupmembers[index].Address] = struct{}{}
			}

		}
	}

	return vg.validateMembers
}

func (vg *ValidateGroup) SelectTxThresholds(number uint64) []common.Address {

	var thresholdMembers []common.Address
	if len(vg.thresholdtxgroupmembers) > 0 && len(vg.thresholdtxgroupmembers) < ValidateLimit {

		for _, member := range vg.thresholdtxgroupmembers {
			thresholdMembers = append(thresholdMembers, member.Address)
		}

	} else {

		epochNumber := uint64(number/blockNumberLimet + 1)

		r := RandFromBytes(encodeBlockNumber(epochNumber))
		size := ValidateLimit
		members := r.RandomPerm(len(vg.thresholdtxgroupmembers), size)

		for _, index := range members {
			thresholdMembers = append(thresholdMembers, vg.thresholdtxgroupmembers[index].Address)
		}

	}

	return thresholdMembers
}

func (vg *ValidateGroup) SelectTxConsensus(number uint64) []common.Address {

	epochNumber := uint64(number / blockNumberLimet)
	if vg.number == epochNumber {
		if len(vg.consensusTxMembers) > 0 {
			return vg.consensusTxMembers
		}
	} else {
		vg.consensusTxMembers = vg.consensusTxMembers[0:0]
	}

	if len(vg.consensusTxAddress) > 0 {
		for k, _ := range vg.consensusTxAddress {
			delete(vg.consensusTxAddress, k)
		}
	}

	fmt.Println("vg.groupmembers(Tx Consensus) :", len(vg.transactiongroupmembers))

	if len(vg.transactiongroupmembers) > 0 {
		if len(vg.transactiongroupmembers) > 0 && len(vg.transactiongroupmembers) < ValidateLimit {

			for _, member := range vg.transactiongroupmembers {
				vg.consensusTxAddress[member.Address] = struct{}{}
				vg.consensusTxMembers = append(vg.consensusTxMembers, member.Address)
			}

		} else {

			r := RandFromBytes(encodeBlockNumber(epochNumber))
			size := ValidateLimit
			members := r.RandomPerm(len(vg.transactiongroupmembers), size)

			for _, index := range members {
				vg.consensusTxMembers = append(vg.consensusTxMembers, vg.transactiongroupmembers[index].Address)
				vg.consensusTxAddress[vg.transactiongroupmembers[index].Address] = struct{}{}
			}

		}
	}

	return vg.consensusTxMembers
}

func (vg *ValidateGroup) IsValidateGroup(addre common.Address) bool {
	_, isTrue := vg.validateAddress[addre]
	return isTrue
}

func (vg *ValidateGroup) IsConsensusGroup(addre common.Address) bool {
	_, isTrue := vg.consensusAddress[addre]
	return isTrue
}

func (vg *ValidateGroup) IsValidate() bool {
	return vg.noValidate
}

func (vg *ValidateGroup) GroupSize() int {
	return len(vg.groupmembers)
}

func GetGroupConst(hash common.Hash, db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(hash.Bytes())
	if len(data) == 0 {

		fmt.Println("**** Invalid GetGroupConst data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid GetGroupConst RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteGroupConst(db lvldb.Putter, hash common.Hash, members GroupMembers) error {
	fmt.Println("WriteGroupConst : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid GetGroupConst RLP data  ", "err", err)
		return err
	}
	return WriteGroupConstRLP(db, hash, data)
}

func WriteGroupConstRLP(db lvldb.Putter, hash common.Hash, rlp rlp.RawValue) error {

	if err := db.Put(hash.Bytes(), rlp); err != nil {
		fmt.Println("Failed to GetGroupConst ", "err", err)
		return err
	}

	return nil
}

func GetValidateGroup(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(groupMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetValidateGroup data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid ValidateGroup RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteValidateGroup(db lvldb.Putter, members GroupMembers) error {
	fmt.Println("WriteValidateGroup : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid ValidateGroup RLP data  ", "err", err)
		return err
	}
	return WriteValidateGroupRLP(db, data)
}

func WriteValidateGroupRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(groupMembersKey, rlp); err != nil {
		fmt.Println("Failed to ValidateGroup ", "err", err)
		return err
	}

	return nil
}

func DeleteValidateGroup(db lvldb.Database) error {
	return db.Delete(groupMembersKey)
}

func GetCoinValidateGroup(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(coingroupMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetCoinValidateGroup data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid GetCoinValidateGroup RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteCoinValidateGroup(db lvldb.Putter, members GroupMembers) error {
	fmt.Println("WriteCoinValidateGroup : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid WriteCoinValidateGroup RLP data  ", "err", err)
		return err
	}
	return WriteCoinValidateGroupRLP(db, data)
}

func WriteCoinValidateGroupRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(coingroupMembersKey, rlp); err != nil {
		fmt.Println("Failed to WriteCoinValidateGroupRLP ", "err", err)
		return err
	}

	return nil
}

func DeleteCoinValidateGroup(db lvldb.Database) error {
	return db.Delete(coingroupMembersKey)
}

//thresholdCoinMembers
func GetThresholdCoinGroup(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(thresholdCoinMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetThresholdCoinGroup data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid GetThresholdCoinGroup RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteThresholdCoinGroup(db lvldb.Putter, members GroupMembers) error {
	fmt.Println("WriteThresholdCoinGroup : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid WriteThresholdCoinGroup RLP data  ", "err", err)
		return err
	}
	return WriteThresholdCoinGroupRLP(db, data)
}

func WriteThresholdCoinGroupRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(thresholdCoinMembersKey, rlp); err != nil {
		fmt.Println("Failed to WriteThresholdCoinGroupRLP ", "err", err)
		return err
	}

	return nil
}

//thresholdMembers
func GetThresholdGroup(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(thresholdMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetThresholdGroup data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid ThresholdGroup RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteThresholdGroup(db lvldb.Putter, members GroupMembers) error {
	fmt.Println("WriteThresholdGroup : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid ThresholdGroup RLP data  ", "err", err)
		return err
	}
	return WriteThresholdGroupRLP(db, data)
}

func WriteThresholdGroupRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(thresholdMembersKey, rlp); err != nil {
		fmt.Println("Failed to ThresholdGroup ", "err", err)
		return err
	}

	return nil
}

func GetStellGroup(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(stellMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetStellGroup data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid StellGroup RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteStellGroup(db lvldb.Putter, members GroupMembers) error {
	fmt.Println("WriteStellGroup : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid StellGroup RLP data  ", "err", err)
		return err
	}
	return WriteStellGroupRLP(db, data)
}

func WriteStellGroupRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(stellMembersKey, rlp); err != nil {
		fmt.Println("Failed to StellGroup ", "err", err)
		return err
	}

	return nil
}

func GetCoinGroup(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(coinMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetCoinGroup data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid CoinGroup RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteCoinGroup(db lvldb.Putter, members GroupMembers) error {

	fmt.Println("WriteCoinGroup : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid WriteCoinGroup RLP data  ", "err", err)
		return err
	}
	return WriteCoinGroupRLP(db, data)
}

func WriteCoinGroupRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(coinMembersKey, rlp); err != nil {
		fmt.Println("Failed to WriteCoinGroup ", "err", err)
		return err
	}
	return nil
}

func GetTransactionGroup(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(transactionMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetTransactionGroup data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid GetTransactionGroup RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteTransactionGroup(db lvldb.Putter, members GroupMembers) error {

	fmt.Println("WriteTransactionGroup : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid WriteTransactionGroup RLP data  ", "err", err)
		return err
	}
	return WriteTransactionGroupRLP(db, data)
}

func WriteTransactionGroupRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(transactionMembersKey, rlp); err != nil {
		fmt.Println("Failed to WriteTransactionGroupRLP ", "err", err)
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////
func GetNewTxMembers(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(newTxMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetNewTxMembers data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid GetNewTxMembers RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteNewTxMembers(db lvldb.Putter, members GroupMembers) error {
	fmt.Println("WriteNewTxMembers : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid WriteNewTxMembers RLP data  ", "err", err)
		return err
	}
	return WriteNewTxMembersRLP(db, data)
}

func WriteNewTxMembersRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(newTxMembersKey, rlp); err != nil {
		fmt.Println("Failed to WriteNewTxMembersRLP ", "err", err)
		return err
	}

	return nil
}

func DeleteNewTxMembers(db lvldb.Database) error {
	return db.Delete(newTxMembersKey)
}

///////////////////////////////////////////////////////////////////////////////////////////////
func GetThresholdTxMembers(db lvldb.Database) (error, int, GroupMembers) {

	var members GroupMembers
	data, _ := db.Get(thresholdTxMembersKey)
	if len(data) == 0 {

		fmt.Println("**** Invalid GetThresholdTxMembers data = 0  ***** ")
		return nil, 0, members
	}

	err := rlp.DecodeBytes(data, &members)
	if err != nil {
		fmt.Println("Invalid GetThresholdTxMembers RLP  ", "err", err)
		return err, 0, members
	}

	return nil, len(members), members
}

func WriteThresholdTxMembers(db lvldb.Putter, members GroupMembers) error {
	fmt.Println("WriteThresholdTxMembers : ", len(members))
	data, err := rlp.EncodeToBytes(members)
	if err != nil {

		fmt.Println("Invalid WriteThresholdTxMembers RLP data  ", "err", err)
		return err
	}
	return WriteThresholdTxMembersRLP(db, data)
}

func WriteThresholdTxMembersRLP(db lvldb.Putter, rlp rlp.RawValue) error {

	if err := db.Put(thresholdTxMembersKey, rlp); err != nil {
		fmt.Println("Failed to WriteThresholdTxMembers RLP ", "err", err)
		return err
	}

	return nil
}

func DeleteThresholdTxMembers(db lvldb.Database) error {
	return db.Delete(thresholdTxMembersKey)
}
