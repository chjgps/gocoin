// +build go1.5
/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2018.5.3
 * Website    : http:www.gocoin.com
 * Function   : define the block structure
**/
package protocol

import (
	"fmt"
	"math/big"
	"sort"

	"gocoin/libraries/common"
)

var (
	EmptyRootHash = DeriveSha(Txs{})
)

const (
	Settlement  = 0
	Publish     = 1
	Recoinages  = 2
	SystemEvent = 3

	ModuleSize = 4
)

type LiquidationReport struct {
	Gas      uint64
	Space_Id *big.Int
}

type LiquidationReports []LiquidationReport

// Header represents a block header in the  blockchain.
type Header struct {
	UncleHash common.Hash
	Number         *big.Int
	IdIndex        *big.Int
	WalletID       uint64
	GroupID        uint32
	Timestamp      *big.Int
	GasUsed        *big.Int
	Version        []byte
	Percent        []byte

	PK              []byte
	Sign            []byte
	LiquidationHash []byte
	ExtraIndex      [ModuleSize]byte
	SpaceRoot       common.Hash
	AccRoot         common.Hash
	GroupRoot       common.Hash
	CoinRoot        common.Hash
	TxHash          common.Hash
	ConsensusGroupHash  common.Hash
	ExtraHash       common.Hash
}

type Block struct {
	Header *Header
	Body   *Block_Body
}

type Block_Body struct {
	Transactions Txs
	ExtraData    []byte
}

func RlpHash(x interface{}) (h common.Hash) {
	return rlpHash(x)
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
func NewBlock(header *Header, body *Block_Body) *Block {
	b := &Block{Header: CopyHeader(header)}

	if body == nil {
		b.Header.TxHash = EmptyRootHash
		b.Header.ExtraHash = common.Hash{}
	} else {

		b.Body = new(Block_Body)
		if len(body.Transactions) > 0 {
			b.Body.Transactions = make(Txs, len(body.Transactions))
			copy(b.Body.Transactions, body.Transactions)
			b.Header.TxHash = DeriveSha(b.Body.Transactions)
		} else {
			b.Header.TxHash = EmptyRootHash
		}
	}

	return b
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{Header: CopyHeader(header)}
}

// CopyHeader creates a deep copy of a block header to prevent side effects from
// modifying a header variable.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Timestamp = new(big.Int); h.Timestamp != nil {
		cpy.Timestamp.Set(h.Timestamp)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if cpy.IdIndex = new(big.Int); h.IdIndex != nil {
		cpy.IdIndex.Set(h.IdIndex)
	}
	if cpy.GasUsed = new(big.Int); h.GasUsed != nil {
		cpy.GasUsed.Set(h.GasUsed)
	}

	cpy.WalletID = h.WalletID
	cpy.GroupID = h.GroupID

	if len(h.Percent) > 0 {
		cpy.Percent = make([]byte, len(h.Percent))
		copy(cpy.Percent, h.Percent)
	}
	if len(h.PK) > 0 {
		cpy.PK = make([]byte, len(h.PK))
		copy(cpy.PK, h.PK)
	}
	if len(h.Sign) > 0 {
		cpy.Sign = make([]byte, len(h.Sign))
		copy(cpy.Sign, h.Sign)
	}
	if len(h.Version) > 0 {
		cpy.Version = make([]byte, len(h.Version))
		copy(cpy.Version, h.Version)
	}

	cpy.CoinRoot = h.CoinRoot
	cpy.GroupRoot = h.GroupRoot
	cpy.ConsensusGroupHash = h.ConsensusGroupHash
	cpy.LiquidationHash = h.LiquidationHash

	return &cpy
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

func (h *Header) HashNoSig() common.Hash {
	return rlpHash([]interface{}{
		h.UncleHash,
		h.Number,
		h.IdIndex,
		h.GroupID,
		h.WalletID,
		h.Timestamp,
		h.GasUsed,
		h.Version,
		h.Percent,
		h.PK,
		h.ExtraIndex,
		h.SpaceRoot,
		h.AccRoot,
		h.GroupRoot,
		h.CoinRoot,
		h.TxHash,
		h.ConsensusGroupHash,
		h.ExtraHash,
		h.LiquidationHash,
	})
}

// TODO: copies
func (b *Block) Transactions() Txs { return b.Body.Transactions }

func (b *Block) Transaction(hash common.Hash) *Tx {
	for _, transaction := range b.Body.Transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *Block) Number() *big.Int      { return new(big.Int).Set(b.Header.Number) }
func (b *Block) Timestamp() *big.Int   { return new(big.Int).Set(b.Header.Timestamp) }
func (b *Block) SpaceIndex() *big.Int  { return new(big.Int).Set(b.Header.IdIndex) }
func (b *Block) WalletIndex() uint64   { return b.Header.WalletID }
func (b *Block) GroupIndex()  uint32   { return b.Header.GroupID }


func (b *Block) GetCoinNum() uint64       { return 0 }
func (b *Block) GetTimestampU64() uint64  { return b.Header.Timestamp.Uint64() }
func (b *Block) GetNumberU64() uint64     { return b.Header.Number.Uint64() }
func (b *Block) GetGasUsedU64() uint64    { return b.Header.GasUsed.Uint64() }
func (b *Block) GetSpaceIndexU64() uint64 { return b.Header.IdIndex.Uint64() }

func (b *Block) GetRoot() common.Hash           { return b.Header.SpaceRoot }
func (b *Block) GetRootAccount() common.Hash    { return b.Header.AccRoot }
func (b *Block) GetParentHash() common.Hash     { return b.Header.UncleHash }
func (b *Block) GetTxHash() common.Hash         { return b.Header.TxHash }
func (b *Block) GetGroupRoot() common.Hash      { return b.Header.GroupRoot }
func (b *Block) GetCoinRoot() common.Hash       { return b.Header.CoinRoot }
func (b *Block) GetConsensusGroupHash() common.Hash { return b.Header.ConsensusGroupHash }
func (b *Block) GetLiquidationHash() []byte     { return common.CopyBytes(b.Header.LiquidationHash) }
func (b *Block) GetPercent() []byte             { return common.CopyBytes(b.Header.Percent) }
func (b *Block) GetVersion() []byte             { return common.CopyBytes(b.Header.Version) }
func (b *Block) GetPK() []byte                  { return common.CopyBytes(b.Header.PK) }

func (b *Block) BlockHead() *Header { return CopyHeader(b.Header) }

// Body returns the non-header content of the block.
func (b *Block) BlockBody() *Block_Body { return b.Body }

func (b *Block) SetPK(pk []byte) {
	if len(pk) > 0 {
		b.Header.PK = make([]byte, len(pk))
		copy(b.Header.PK, pk)
	}
}

func (b *Block) SetSigner(sig []byte) {
	if len(sig) > 0 {
		b.Header.Sign = make([]byte, len(sig))
		copy(b.Header.Sign, sig)
	}
}

// WithBody returns a new block with the given transaction.
func (b *Block) WithBody(transactions Txs, extradata []byte) *Block {
	block := &Block{
		Header: CopyHeader(b.Header),
		Body:   new(Block_Body),
	}

	if len(transactions) > 0 {
		block.Body.Transactions = make(Txs, len(transactions))
		copy(block.Body.Transactions, transactions)
		block.Header.TxHash = DeriveSha(block.Body.Transactions)
	} else {
		block.Header.TxHash = EmptyRootHash
	}

	if len(extradata) > 0 {
		block.Body.ExtraData = make([]byte, len(extradata))
		copy(block.Body.ExtraData, extradata)
	}

	return block
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *Block) Hash() common.Hash {
	return b.Header.Hash()
}

func (b *Block) String() string {
	str := fmt.Sprintf(`Block(#%v): {
		header: %v
		Transactions:%v
		extraData:%v} `, b.Number(), b.Header, b.Body.Transactions, b.Body.ExtraData)
	return str
}

func (h *Header) String() string {
	return fmt.Sprintf(`Header(%x):
[
	ParentHash:	    %x
	SpaceRoot:		    %x
	AccRoot:          %x
    GroupRoot         %x
    CoinRoot          %x
	TxHash:		        %x
	Number:		        %v
	IdIndex:	        %v
	Timestamp:		    %v
	WalletIndex:       %v
    GroupIndex:        %v
	GasUsed            %v
	Percent            %s
	PK                 %s
	ExtraIndex         %v
	ExtraHash          %x
    ConsensusGroupHash     %x
	LiquidationHash    %x
	Version:		     %s
]`, h.Hash(), h.UncleHash, h.SpaceRoot, h.AccRoot,h.GroupRoot,h.CoinRoot, h.TxHash, h.Number, h.IdIndex, h.Timestamp, h.WalletID, h.GroupID, h.GasUsed, h.Percent, h.PK, h.ExtraIndex, h.ExtraHash,h.ConsensusGroupHash, h.LiquidationHash, h.Version)
}

type Blocks []*Block

type BlockBy func(b1, b2 *Block) bool

func (self BlockBy) Sort(blocks Blocks) {
	bs := blockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type blockSorter struct {
	blocks Blocks
	by     func(b1, b2 *Block) bool
}

func (self blockSorter) Len() int { return len(self.blocks) }
func (self blockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}
func (self blockSorter) Less(i, j int) bool { return self.by(self.blocks[i], self.blocks[j]) }

func Number(b1, b2 *Block) bool { return b1.Header.Number.Cmp(b2.Header.Number) < 0 }
