/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : block chain database
**/

package space

import (
	"bytes"
	"encoding/binary"

	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/log"
	"gocoin/libraries/rlp"
)

// DatabaseReader wraps the Get method of a backing data store.
type DatabaseReader interface {
	Get(key []byte) (value []byte, err error)
}

// DatabaseDeleter wraps the Delete method of a backing data store.
type DatabaseDeleter interface {
	Delete(key []byte) error
}

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")
	headFastKey   = []byte("LastFast")

	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`).
	headerPrefix    = []byte("sh") // headerPrefix + num (uint64 big endian) + hash -> header
	numSuffix       = []byte("sn") // headerPrefix + num (uint64 big endian) + numSuffix -> hash
	blockHashPrefix = []byte("sH") // blockHashPrefix + hash -> num (uint64 big endian)
	bodyPrefix      = []byte("sb") // bodyPrefix + num (uint64 big endian) + hash -> block body
	lookupPrefix    = []byte("sl") // lookupPrefix + hash -> transaction lookup metadata

	liquidationfix  = []byte("lr")
	oldTxMetaSuffix = []byte{0x01}
)

// TxLookupEntry is a positional metadata to help looking up the data content of
// a transaction  given only its hash.
type TxLookupEntry struct {
	BlockHash  common.Hash
	BlockIndex uint64
	Index      uint64
}

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// GetCanonicalHash retrieves a hash assigned to a canonical block number.
func GetCanonicalHash(db DatabaseReader, number uint64) common.Hash {
	data, _ := db.Get(append(append(headerPrefix, encodeBlockNumber(number)...), numSuffix...))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// missingNumber is returned by GetBlockNumber if no header with the
// given block hash has been stored in the database
const missingNumber = uint64(0xffffffffffffffff)

// GetBlockNumber returns the block number assigned to a block hash
// if the corresponding header is present in the database
func GetBlockNumber(db DatabaseReader, hash common.Hash) uint64 {
	data, _ := db.Get(append(blockHashPrefix, hash.Bytes()...))
	if len(data) != 8 {
		return missingNumber
	}
	return binary.BigEndian.Uint64(data)
}

// GetHeadHeaderHash retrieves the hash of the current canonical head block's
// header. The difference between this and GetHeadBlockHash is that whereas the
// last block hash is only updated upon a full block import, the last header
// hash is updated already at header import, allowing head tracking for the
// light synchronization mechanism.
func GetHeadHeaderHash(db DatabaseReader) common.Hash {
	data, _ := db.Get(headHeaderKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadBlockHash retrieves the hash of the current canonical head block.
func GetHeadBlockHash(db DatabaseReader) common.Hash {
	data, _ := db.Get(headBlockKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadFastBlockHash retrieves the hash of the current canonical head block during
// fast synchronization. The difference between this and GetHeadBlockHash is that
// whereas the last block hash is only updated upon a full block import, the last
// fast hash is updated when importing pre-processed blocks.
func GetHeadFastBlockHash(db DatabaseReader) common.Hash {
	data, _ := db.Get(headFastKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeaderRLP retrieves a block header in its raw RLP database encoding, or nil
// if the header's not found.
func GetHeaderRLP(db DatabaseReader, hash common.Hash, number uint64) rlp.RawValue {
	data, _ := db.Get(headerKey(hash, number))
	return data
}

// GetHeader retrieves the block header corresponding to the hash, nil if none
// found.
func GetHeader(db DatabaseReader, hash common.Hash, number uint64) *protocol.Header {
	data := GetHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	header := new(protocol.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		log.Error("Invalid block header RLP", "hash", hash, "err", err)
		return nil
	}
	return header
}

// GetBodyRLP retrieves the block body (transactions and uncles) in RLP encoding.
func GetBodyRLP(db DatabaseReader, hash common.Hash, number uint64) rlp.RawValue {
	data, _ := db.Get(blockBodyKey(hash, number))
	return data
}

func headerKey(hash common.Hash, number uint64) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

func blockBodyKey(hash common.Hash, number uint64) []byte {
	return append(append(bodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// GetBody retrieves the block body (transactons, uncles) corresponding to the
// hash, nil if none found.
func GetBody(db DatabaseReader, hash common.Hash, number uint64) *protocol.Block_Body {
	data := GetBodyRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	body := new(protocol.Block_Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		log.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}
	return body
}

// GetBlock retrieves an entire block corresponding to the hash, assembling it
// back from the stored header and body. If either the header or body could not
// be retrieved nil is returned.
//
// Note, due to concurrent download of header and block body the header and thus
// canonical hash can be stored in the database but the body data not (yet).
func GetBlock(db DatabaseReader, hash common.Hash, number uint64) *protocol.Block {
	// Retrieve the block header and body contents
	header := GetHeader(db, hash, number)
	if header == nil {
		return nil
	}
	body := GetBody(db, hash, number)
	if body == nil {
		return nil
	}
	// Reassemble the block and return
	return protocol.NewBlock(header, body)
}

// GetTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hash.
func GetTxLookupEntry(db DatabaseReader, hash common.Hash) (common.Hash, uint64, uint64) {
	// Load the positional metadata from disk and bail if it fails
	data, _ := db.Get(append(lookupPrefix, hash.Bytes()...))
	if len(data) == 0 {
		return common.Hash{}, 0, 0
	}
	// Parse and return the contents of the lookup entry
	var entry TxLookupEntry
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		log.Error("Invalid lookup entry RLP", "hash", hash, "err", err)
		return common.Hash{}, 0, 0
	}
	return entry.BlockHash, entry.BlockIndex, entry.Index
}

// GetTransaction retrieves a specific transaction from the database, along with
// its added positional metadata.
func GetTransaction(db DatabaseReader, hash common.Hash) (*protocol.Tx, common.Hash, uint64, uint64) {
	// Retrieve the lookup metadata and resolve the transaction from the body
	//blockHash, blockNumber, txIndex := GetTxLookupEntry(db, hash)
	/*
		if blockHash != (common.Hash{}) {
			body := GetBody(db, blockHash, blockNumber)
			if body == nil || len(body.Transactions) <= int(txIndex) {
				log.Error("Transaction referenced missing", "number", blockNumber, "hash", blockHash, "index", txIndex)
				return nil, common.Hash{}, 0, 0
			}
			return body.Transactions[txIndex], blockHash, blockNumber, txIndex
		}*/

	// Old transaction representation, load the transaction and it's metadata separately
	data, _ := db.Get(hash.Bytes())
	if len(data) == 0 {
		return nil, common.Hash{}, 0, 0
	}
	var tx protocol.Tx
	if err := rlp.DecodeBytes(data, &tx); err != nil {
		return nil, common.Hash{}, 0, 0
	}
	// Retrieve the blockchain positional metadata
	data, _ = db.Get(append(hash.Bytes(), oldTxMetaSuffix...))
	if len(data) == 0 {
		return nil, common.Hash{}, 0, 0
	}
	var entry TxLookupEntry
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		return nil, common.Hash{}, 0, 0
	}
	return &tx, entry.BlockHash, entry.BlockIndex, entry.Index
}

// WriteCanonicalHash stores the canonical hash for the given block number.
func WriteCanonicalHash(db lvldb.Putter, hash common.Hash, number uint64) error {
	key := append(append(headerPrefix, encodeBlockNumber(number)...), numSuffix...)
	if err := db.Put(key, hash.Bytes()); err != nil {
		log.Crit("Failed to store number to hash mapping", "err", err)
	}
	return nil
}

// WriteHeadHeaderHash stores the head header's hash.
func WriteHeadHeaderHash(db lvldb.Putter, hash common.Hash) error {
	if err := db.Put(headHeaderKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last header's hash", "err", err)
	}
	return nil
}

// WriteHeadBlockHash stores the head block's hash.
func WriteHeadBlockHash(db lvldb.Putter, hash common.Hash) error {
	if err := db.Put(headBlockKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last block's hash", "err", err)
	}
	return nil
}

// WriteHeadFastBlockHash stores the fast head block's hash.
func WriteHeadFastBlockHash(db lvldb.Putter, hash common.Hash) error {
	if err := db.Put(headFastKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last fast block's hash", "err", err)
	}
	return nil
}

// WriteHeader serializes a block header into the database.
func WriteHeader(db lvldb.Putter, header *protocol.Header) error {
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		return err
	}
	hash := header.Hash().Bytes()
	num := header.Number.Uint64()
	encNum := encodeBlockNumber(num)
	key := append(blockHashPrefix, hash...)
	if err := db.Put(key, encNum); err != nil {
		log.Crit("Failed to store hash to number mapping", "err", err)
	}
	key = append(append(headerPrefix, encNum...), hash...)
	if err := db.Put(key, data); err != nil {
		log.Crit("Failed to store header", "err", err)
	}
	return nil
}

// WriteBody serializes the body of a block into the database.
func WriteBody(db lvldb.Putter, hash common.Hash, number uint64, body *protocol.Block_Body) error {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		return err
	}
	return WriteBodyRLP(db, hash, number, data)
}

// WriteBodyRLP writes a serialized body of a block into the database.
func WriteBodyRLP(db lvldb.Putter, hash common.Hash, number uint64, rlp rlp.RawValue) error {
	key := append(append(bodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
	if err := db.Put(key, rlp); err != nil {
		log.Crit("Failed to store block body", "err", err)
	}
	return nil
}

// WriteBlock serializes a block into the database, header and body separately.
func WriteBlock(db lvldb.Putter, block *protocol.Block) error {
	// Store the body first to retain database consistency
	if err := WriteBody(db, block.Hash(), block.GetNumberU64(), block.BlockBody()); err != nil {
		return err
	}
	// Store the header too, signaling full block ownership
	if err := WriteHeader(db, block.BlockHead()); err != nil {
		return err
	}
	return nil
}

// WriteTxLookupEntries stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntries(db lvldb.Putter, block *protocol.Block) error {
	// Iterate over each transaction and encode its metadata
	for i, tx := range block.Transactions() {
		entry := TxLookupEntry{
			BlockHash:  block.Hash(),
			BlockIndex: block.GetNumberU64(),
			Index:      uint64(i),
		}
		data, err := rlp.EncodeToBytes(entry)
		if err != nil {
			return err
		}
		if err := db.Put(append(lookupPrefix, tx.Hash().Bytes()...), data); err != nil {
			return err
		}
	}
	return nil
}

// GetTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hash.
func GetLiquidation(db DatabaseReader, hash common.Hash) ([]byte, error) {
	// Load the positional metadata from disk and bail if it fails
	return db.Get(append(liquidationfix, hash.Bytes()...))
}

func WriteLiquidationHash(db lvldb.Putter, hash common.Hash, data []byte) error {

	if err := db.Put(append(liquidationfix, hash.Bytes()...), data); err != nil {
		return err
	}

	return nil
}

func DeleteLiquidationHash(db DatabaseDeleter, hash common.Hash) error {
	return db.Delete(append(liquidationfix, hash.Bytes()...))
}

// DeleteCanonicalHash removes the number to hash canonical mapping.
func DeleteCanonicalHash(db DatabaseDeleter, number uint64) {
	db.Delete(append(append(headerPrefix, encodeBlockNumber(number)...), numSuffix...))
}

// DeleteHeader removes all block header data associated with a hash.
func DeleteHeader(db DatabaseDeleter, hash common.Hash, number uint64) {
	db.Delete(append(blockHashPrefix, hash.Bytes()...))
	db.Delete(append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...))
}

// DeleteBody removes all block body data associated with a hash.
func DeleteBody(db DatabaseDeleter, hash common.Hash, number uint64) {
	db.Delete(append(append(bodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...))
}

// DeleteBlock removes all block data associated with a hash.
func DeleteBlock(db DatabaseDeleter, hash common.Hash, number uint64) {
	DeleteHeader(db, hash, number)
	DeleteBody(db, hash, number)
}

// DeleteTxLookupEntry removes all transaction data associated with a hash.
func DeleteTxLookupEntry(db DatabaseDeleter, hash common.Hash) {
	db.Delete(append(lookupPrefix, hash.Bytes()...))
}

// GetBlockChainVersion reads the version number from db.
func GetBlockChainVersion(db DatabaseReader) int {
	var vsn uint
	enc, _ := db.Get([]byte("BlockchainVersion"))
	rlp.DecodeBytes(enc, &vsn)
	return int(vsn)
}

// WriteBlockChainVersion writes vsn as the version number to db.
func WriteBlockChainVersion(db lvldb.Putter, vsn int) {
	enc, _ := rlp.EncodeToBytes(uint(vsn))
	db.Put([]byte("BlockchainVersion"), enc)
}

// FindCommonAncestor returns the last common ancestor of two block headers
func FindCommonAncestor(db DatabaseReader, a, b *protocol.Header) *protocol.Header {
	for bn := b.Number.Uint64(); a.Number.Uint64() > bn; {
		a = GetHeader(db, a.UncleHash, a.Number.Uint64()-1)
		if a == nil {
			return nil
		}
	}
	for an := a.Number.Uint64(); an < b.Number.Uint64(); {
		b = GetHeader(db, b.UncleHash, b.Number.Uint64()-1)
		if b == nil {
			return nil
		}
	}
	for a.Hash() != b.Hash() {
		a = GetHeader(db, a.UncleHash, a.Number.Uint64()-1)
		if a == nil {
			return nil
		}
		b = GetHeader(db, b.UncleHash, b.Number.Uint64()-1)
		if b == nil {
			return nil
		}
	}
	return a
}
