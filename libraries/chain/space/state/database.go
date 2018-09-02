/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.21
 * Website    : http:www.gocoin.com
 * Function   : state database  interface
**/

package state

import (
	"gocoin/libraries/chain/space/trie"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
)

type Trie interface {
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	CommitTo(trie.DatabaseWriter) (common.Hash, error)
	Hash() common.Hash
	NodeIterator(startKey []byte) trie.NodeIterator
	GetKey([]byte) []byte
}

type Database interface {
	OpenSpaceTrie(root common.Hash) (Trie, error)
	OpenAccountTrie(root common.Hash) (Trie, error)
}

func NewDatabase(db lvldb.Database) Database {
	return &cachingDB{db: db}
}

type cachingDB struct {
	db lvldb.Database
}

func (db *cachingDB) OpenSpaceTrie(root common.Hash) (Trie, error) {
	return trie.NewSecure(root, db.db, 0)
}

func (db *cachingDB) OpenAccountTrie(root common.Hash) (Trie, error) {
	return trie.NewSecure(root, db.db, 0)
}
