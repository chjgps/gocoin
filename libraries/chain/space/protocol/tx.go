/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.12.3
 * Website    : http:www.gocoin.com
 * Function   : transaction
**/

package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"

	"gocoin/libraries/common"
	"gocoin/libraries/crypto"
	"gocoin/libraries/crypto/sha3"
	"gocoin/libraries/rlp"
	"time"
)

const (
	TX_BASE      = 0x0A
	TX_PROPOSE   = TX_BASE + 1
	TX_CONSENSUS = TX_BASE + 2
	TX_TRADE     = TX_BASE + 3
)

const (
	TX_NORMAL  = TX_BASE + 10
	TX_WARNING = TX_BASE + 11
	TX_INFO    = TX_BASE + 12
	TX_ERROR   = TX_BASE + 13
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
	errNoSigner   = errors.New("missing signing methods")
)

type Tx struct {
	data Txstruct

	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type Txstruct struct {
	Cost      *big.Int        `json:"cost"      gencodec:"required"` // cost
	Recipient *common.Address `json:"to"       rlp:"nil"`            // to
	Amount    *big.Int        `json:"value"    gencodec:"required"`  // transfer
	Txtime    uint64
	Payload   []byte `json:"input"    gencodec:"required"` // input

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	Hash *common.Hash `json:"hash" rlp:"-"`
}

type Attr struct {
	K []byte
	V []byte
}

// TxPreEvent is posted when a transaction enters the transaction pool.
type TxPreEvent struct{ Transaction *Tx }

const (
	idtSpaceCreation   = iota + 1
	idtRecharge        //2
	idtAttrSet         //3
	idCreateGroup      //4
	IdPublishCoin      //5
	IdEpochTx          //6
	IdEpochSigTx       //7
	IdApplyGroup       //8
	IdStellReport      //9
	IdStellGroup       //10
	IdCoinGroup        //11
	IdTransactionGroup //12
	IdCreateContractTx //13
	IdRunContractTx    //14
)

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func NewSettleHashTx(datahash []byte) *Tx {

	var ptype byte
	ptype = IdStellReport
	b := new(bytes.Buffer)
	b.WriteByte(ptype)
	b.Write(datahash)

	return newTx(&common.Address{}, big.NewInt(0), big.NewInt(0), b.Bytes())
}

func TxbyData(data Txstruct) *Tx {
	tx := &Tx{
		data: data,
	}
	return tx
}

func NewTx(to common.Address, amount, cost *big.Int, data []byte) *Tx {
	return newTx(&to, amount, cost, data)
}

func NewSpaceCreation(cost *big.Int, bindAddr common.Address) *Tx {

	var ptype byte
	ptype = idtSpaceCreation

	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	return newTx(&bindAddr, big.NewInt(0), cost, b.Bytes())
}

func CreateContractTx(cost *big.Int, data []byte) *Tx {
	var ptype byte
	ptype = IdCreateContractTx
	b := new(bytes.Buffer)
	b.WriteByte(ptype)
	b.Write(data)

	return newTx(&common.Address{}, big.NewInt(0), cost, b.Bytes())
}
func RunContractTx(to common.Address, amount, cost *big.Int, data []byte) *Tx {
	var ptype byte
	ptype = IdRunContractTx
	b := new(bytes.Buffer)
	b.WriteByte(ptype)
	b.Write(data)

	return newTx(&to, amount, cost, b.Bytes())
}

func ApplyStellGroup(to common.Address, amount, cost *big.Int) *Tx {
	var ptype byte
	ptype = IdStellGroup
	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	return newTx(&to, amount, cost, b.Bytes())
}

func ApplyCoinGroup(to common.Address, amount, cost *big.Int) *Tx {
	var ptype byte
	ptype = IdCoinGroup
	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	return newTx(&to, amount, cost, b.Bytes())
}

func ApplyTransactionGroup(to common.Address, amount, cost *big.Int) *Tx {
	var ptype byte
	ptype = IdTransactionGroup
	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	return newTx(&to, amount, cost, b.Bytes())
}

func ApplyValidateGroup(to common.Address, amount, cost *big.Int) *Tx {
	var ptype byte
	ptype = IdApplyGroup
	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	return newTx(&to, amount, cost, b.Bytes())
}

func NewRecharge(to common.Address, amount, cost *big.Int) *Tx {

	var ptype byte
	ptype = idtRecharge
	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	return newTx(&to, amount, cost, b.Bytes())
}

func NewAttrSet(to common.Address, cost *big.Int, attr Attr) *Tx {

	data, _ := rlp.EncodeToBytes(attr)
	var ptype byte
	ptype = idtAttrSet
	b := new(bytes.Buffer)
	b.WriteByte(ptype)
	b.Write(data)

	return newTx(&to, big.NewInt(0), cost, b.Bytes())
}

func NewCreateGroup(gaddr common.Address) *Tx {

	var ptype byte
	ptype = idCreateGroup
	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	return newTx(&gaddr, big.NewInt(0), big.NewInt(0), b.Bytes())
}

func NewPublishCon(amount *big.Int, cost *big.Int, data []byte) *Tx {
	var ptype byte
	ptype = IdPublishCoin
	b := new(bytes.Buffer)
	b.WriteByte(ptype)
	b.Write(data)

	return newTx(&common.Address{}, amount, cost, b.Bytes())
}

func NewEpochTx(cost *big.Int, data []byte) *Tx {
	var ptype byte
	ptype = IdEpochTx
	b := new(bytes.Buffer)
	b.WriteByte(ptype)

	n := uint32(len(data))
	datasize := make([]byte, 4)
	binary.BigEndian.PutUint32(datasize[:], n)
	b.Write(datasize)
	b.Write(data)

	return newTx(&common.Address{}, big.NewInt(0), cost, b.Bytes())
}

func newTx(to *common.Address, amount, cost *big.Int, data []byte) *Tx {

	d := Txstruct{
		Recipient: to,
		Payload:   data,
		Amount:    new(big.Int),
		Cost:      new(big.Int),
		Txtime:    uint64(time.Now().Unix()),

		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if cost != nil {
		d.Cost.Set(cost)
	}

	return &Tx{data: d}
}

// EncodeRLP implements rlp.Encoder
func (tx *Tx) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Tx) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

func (tx *Tx) AddEpochSig(sig []byte) {

	var ptype byte
	ptype = IdEpochSigTx
	b := new(bytes.Buffer)
	b.WriteByte(ptype)
	data := tx.Data()
	b.Write(data[1:])

	n := uint32(len(sig))
	sigsize := make([]byte, 4)
	binary.BigEndian.PutUint32(sigsize[:], n)
	b.Write(sigsize)
	b.Write(sig)

	copy(tx.data.Payload, b.Bytes())
}

func (tx *Tx) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

func (tx *Tx) UnmarshalJSON(input []byte) error {
	var dec Txstruct
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	var V byte

	V = byte(dec.V.Uint64() - 27)

	if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
		return ErrInvalidSig
	}
	*tx = Tx{data: dec}
	return nil
}

func (tx *Tx) Data() []byte          { return common.CopyBytes(tx.data.Payload) }
func (tx *Tx) DataHash() common.Hash { return rlpHash(tx.data.Payload) }
func (tx *Tx) Cost() *big.Int        { return new(big.Int).Set(tx.data.Cost) }
func (tx *Tx) Value() *big.Int       { return new(big.Int).Set(tx.data.Amount) }
func (tx *Tx) Time() uint64          { return tx.data.Txtime }
func (tx *Tx) TxData() *Txstruct {
	txdata := &Txstruct{
		Recipient: tx.data.Recipient,
		Payload:   tx.data.Payload,
		Amount:    tx.data.Amount,
		Cost:      tx.data.Cost,
		Txtime:    tx.data.Txtime,

		V: tx.data.V,
		R: tx.data.R,
		S: tx.data.S,
	}

	return txdata
}

func (tx *Tx) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	} else {
		to := *tx.data.Recipient
		return &to
	}
}

func (tx *Tx) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

type writeCounter common.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func (tx *Tx) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

func (tx *Tx) WithSignature(signer Signer, sig []byte) (*Tx, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Tx{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

func (tx *Tx) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

func (tx *Tx) String() string {
	var from, to string
	if tx.data.V != nil {
		signer := MakeSigner()
		if f, err := Sender(signer, tx); err != nil { // derive but don't cache
			from = "[invalid sender: invalid sig]"
		} else {
			from = fmt.Sprintf("%x", f[:])
		}
	} else {
		from = "[invalid sender: nil V field]"
	}

	if tx.data.Recipient == nil {
		to = "[contract creation]"
	} else {
		to = fmt.Sprintf("%x", tx.data.Recipient[:])
	}
	enc, _ := rlp.EncodeToBytes(&tx.data)
	return fmt.Sprintf(`
	TX(%x)
	Space: %v
	From:     %s
	To:       %s
	Cost:     %#x
	Value:    %#x
	Data:     0x%x
	V:        %#x
	R:        %#x
	S:        %#x
	Hex:      %x
`,
		tx.Hash(),
		tx.data.Recipient == nil,
		from,
		to,
		tx.data.Cost,
		tx.data.Amount,
		tx.data.Payload,
		tx.data.V,
		tx.data.R,
		tx.data.S,
		enc,
	)
}

type Txs []*Tx

// Len returns the length of s
func (s Txs) Len() int { return len(s) }

func (s Txs) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s Txs) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

func TxDifference(a, b Txs) (keep Txs) {
	keep = make(Txs, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

type TxByCost Txs

func (s TxByCost) Len() int           { return len(s) }
func (s TxByCost) Less(i, j int) bool { return s[i].data.Cost.Cmp(s[j].data.Cost) > 0 }
func (s TxByCost) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByCost) Push(x interface{}) {
	*s = append(*s, x.(*Tx))
}

func (s *TxByCost) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

type Message struct {
	Hash   *common.Hash
	To     *common.Address
	From   *common.Address
	Cost   *big.Int
	Amount *big.Int
	Data   []byte
}
