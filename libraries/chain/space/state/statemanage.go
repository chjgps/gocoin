/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.21
 * Website    : http:www.gocoin.com
 * Function   : space state and account state to database
**/

package state

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"gocoin/libraries/chain/space/trie"
	"gocoin/libraries/common"
	"gocoin/libraries/log"
	"gocoin/libraries/rlp"
)

// Errors
var (
	errSpaceId      = errors.New("Space trie does not exist SpaceId")
	errAddress      = errors.New("Account trie does not exist Address")
	errSpaceIdExist = errors.New("Space trie does  exist SpaceId")
	errAddressExist = errors.New("Account trie does  exist Address")
)

type StateDB struct {
	db          Database
	trieSpace   Trie
	trieAccount Trie

	spaceObjects   map[uint64]*spaceObject
	accountObjects map[common.Address]*accountObject

	spaceObjectsDirty   map[uint64]struct{}
	accountObjectsDirty map[common.Address]struct{}

	dbErr error

	lock sync.Mutex
}

func New(rootSpace common.Hash, rootAccount common.Hash, db Database) (*StateDB, error) {
	trSpace, err := db.OpenSpaceTrie(rootSpace)
	if err != nil {
		return nil, err
	}

	trAccount, err := db.OpenAccountTrie(rootAccount)
	if err != nil {
		return nil, err
	}

	return &StateDB{
		db:                  db,
		trieSpace:           trSpace,
		trieAccount:         trAccount,
		spaceObjects:        make(map[uint64]*spaceObject),
		accountObjects:      make(map[common.Address]*accountObject),
		spaceObjectsDirty:   make(map[uint64]struct{}),
		accountObjectsDirty: make(map[common.Address]struct{}),
	}, nil
}

func (self *StateDB) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *StateDB) Error() error {
	return self.dbErr
}

func (self *StateDB) GetBalance(addr common.Address) (*big.Int, error) {
	accountObject, err := self.getAccountObject(addr)
	if accountObject != nil {
		return accountObject.GetBalance(), nil
	}
	return common.Big0, err
}

func (self *StateDB) GetWalletID(addr common.Address) (uint64, error) {
	accountObject, err := self.getAccountObject(addr)
	if accountObject != nil {
		return accountObject.GetWalletID(), nil
	}
	return 0, err
}

func (self *StateDB) AddBalance(addr common.Address, amount *big.Int) error {
	accountObject, err := self.getAccountObject(addr)
	if accountObject != nil {
		accountObject.AddBalance_gas(amount)
	}

	return err
}

func (self *StateDB) SubBalance(addr common.Address, amount *big.Int) error {
	accountObject, err := self.getAccountObject(addr)
	if accountObject != nil {
		accountObject.SubBalance_gas(amount)
	}

	return err
}

func (self *StateDB) SetBalance(addr common.Address, amount *big.Int) error {
	accountObject, err := self.getAccountObject(addr)
	if accountObject != nil {
		accountObject.SetBalance_gas(amount)
	}
	return err
}

func (self *StateDB) GetSpaceBalanceByAddress(addr common.Address) (*big.Int, error) {
	spaceObject, err := self.getSpaceObjectByAddr(addr)
	if spaceObject != nil {
		return spaceObject.Balance(), nil
	}
	return common.Big0, err
}

func (self *StateDB) GetSpaceBalanceByID(id *big.Int) (*big.Int, error) {
	spaceObject, err := self.getSpaceObject(id)
	if spaceObject != nil {
		return spaceObject.Balance(), nil
	}
	return common.Big0, err
}

func (self *StateDB) AddSpaceBalanceByID(id *big.Int, amount *big.Int) error {

	spaceObject, err := self.getSpaceObject(id)
	if spaceObject != nil {
		spaceObject.AddBalance(amount)
	}

	return err
}

func (self *StateDB) SubSpaceBalanceByID(id *big.Int, amount *big.Int) error {

	spaceObject, err := self.getSpaceObject(id)
	if spaceObject != nil {
		spaceObject.SubBalance(amount)
	}

	return err
}

func (self *StateDB) AddSpaceBalanceByAddress(addr common.Address, amount *big.Int) error {
	spaceObject, err := self.getSpaceObjectByAddr(addr)
	if spaceObject != nil {
		spaceObject.AddBalance(amount)
	}

	return err
}

func (self *StateDB) SubSpaceBalanceByAddress(addr common.Address, amount *big.Int) error {
	spaceObject, err := self.getSpaceObjectByAddr(addr)
	if spaceObject != nil {
		spaceObject.SubBalance(amount)
	}

	return err
}

func (self *StateDB) SetSpaceBalanceByAddress(addr common.Address, amount *big.Int) error {
	spaceObject, err := self.getSpaceObjectByAddr(addr)
	if spaceObject != nil {
		spaceObject.SetBalance(amount)
	}

	return err
}

func (self *StateDB) SetAttribute(addr common.Address, key, value []byte) error {
	accountObject, err := self.GetAccountObject(addr)
	if accountObject != nil {
		accountObject.SetAttribute(key, value)
	}

	return err
}

func (self *StateDB) DelAttribute(addr common.Address, key []byte) error {
	accountObject, err := self.GetAccountObject(addr)
	if accountObject != nil {
		accountObject.DelAttribute(key)
	}

	return err
}

func (self *StateDB) GetAttribute(addr common.Address, key []byte) ([]byte, error) {
	var value []byte
	accountObject, err := self.GetAccountObject(addr)
	if accountObject != nil {
		value = accountObject.GetAttribute(key)
	}

	return value, err
}

// MarkAccountObjectsDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *StateDB) MarkAccountObjectsDirty(addr common.Address) {
	self.accountObjectsDirty[addr] = struct{}{}
}

func (self *StateDB) getAccountObject(addr common.Address) (*accountObject, error) {

	//fmt.Println("getAccountObject : ",addr)
	obj, flag := self.accountObjects[addr]
	if flag {
		return obj, nil
	}

	enc, err := self.trieAccount.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil, err
	}

	var acc Account
	if err := rlp.DecodeBytes(enc, &acc); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil, err
	}

	obj = newAccountObject(addr, acc, self.MarkAccountObjectsDirty)
	self.setAccountObject(obj)
	//fmt.Println("accountObject",obj)
	return obj, nil
}

func (self *StateDB) setAccountObject(object *accountObject) {
	self.accountObjects[object.Address()] = object
}

func (self *StateDB) GetAccountObject(addr common.Address) (*accountObject, error) {

	obj, err := self.getAccountObject(addr)

	if (err != nil) || (obj == nil) {
		return nil, err
	}

	return obj, err
}

func (self *StateDB) createAccountObject(id uint64, addr common.Address) (*accountObject, error) {
	prev, err := self.getAccountObject(addr)
	if err == nil {
		newobj := newAccountObject(addr, Account{BindSpaceId: new(big.Int), Balance: new(big.Int),WalletID:id}, self.MarkAccountObjectsDirty)
		self.setAccountObject(newobj)
		return newobj, nil
	}

	return prev, errAddressExist
}

func (self *StateDB) getSpaceObjectByAddr(addr common.Address) (*spaceObject, error) {
	accountObject, err := self.getAccountObject(addr)
	if accountObject != nil {
		spacecObject, err := self.getSpaceObject(accountObject.SpaceID())
		return spacecObject, err
	}

	return nil, err
}

// MarkAccountObjectsDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *StateDB) MarkSpaceObjectsDirty(id uint64) {
	self.spaceObjectsDirty[id] = struct{}{}
}

func (self *StateDB) getSpaceObject(id *big.Int) (*spaceObject, error) {
	if obj := self.spaceObjects[id.Uint64()]; obj != nil {
		return obj, nil
	}

	enc, err := self.trieSpace.TryGet(id.Bytes())
	if len(enc) == 0 {
		self.setError(err)
		return nil, err
	}

	var space Space
	if err := rlp.DecodeBytes(enc, &space); err != nil {
		log.Error("Failed to decode state object", "id", id, "err", err)
		return nil, err
	}

	obj := newSpaceObject(id, space, self.MarkSpaceObjectsDirty)
	self.setSpaceObject(obj)
	return obj, nil
}

func (self *StateDB) setSpaceObject(object *spaceObject) {
	self.spaceObjects[object.Id().Uint64()] = object
}

func (self *StateDB) GetSpaceObject(id *big.Int) (*spaceObject, error) {
	return self.getSpaceObject(id)
}

func (self *StateDB) CreateSpaceObject(id *big.Int) (*spaceObject, error) {

	spaceObject, _ := self.getSpaceObject(id)

	if spaceObject == nil {
		newobj := newSpaceObject(id, Space{Balance: new(big.Int), Nonce: 0, BindAddr: make([]common.Address, 0)}, self.MarkSpaceObjectsDirty)
		self.setSpaceObject(newobj)
		return newobj, nil
	}

	return spaceObject, errSpaceIdExist
}

func (self *StateDB) BindAddr(id *big.Int, walletid uint64, bindAddr common.Address) uint64 {
	space, err := self.getSpaceObject(id)
	if err == nil {
		acc, err := self.GetAccountObject(bindAddr)
		if err == nil {
			self.associateSpaceAndAccount(space, acc)
		} else {
			newacc, _ := self.createAccountObject(walletid,bindAddr)
			if newacc != nil {
				self.associateSpaceAndAccount(space, newacc)
				return  walletid + 1
			}
		}
	}

	return  walletid
}

func (self *StateDB) UnbindAddr(id *big.Int, bindAddr common.Address) {
	space, _ := self.getSpaceObject(id)
	if space != nil {
		acc, err := self.getAccountObject(bindAddr)
		if err == nil {
			self.unassociateSpaceAndAccount(space, acc)
		}
	}
}

func (self *StateDB) associateSpaceAndAccount(space *spaceObject, account *accountObject) {

	if account != nil && space != nil {
		if !account.empty() {
			prev, err := self.getSpaceObject(account.data.BindSpaceId)
			if err == nil {
				self.unassociateSpaceAndAccount(prev, account)
			}
		}

		account.SetSpaceID(space.Id())
		space.AppendAddress(account.Address())
	}
}

func (self *StateDB) unassociateSpaceAndAccount(space *spaceObject, account *accountObject) {
	if account != nil && space != nil {
		account.SetSpaceID(new(big.Int))
		space.RemoveAddress(account.Address())
	}
}

func (self *StateDB) CreateSpaceAndBindAddr(id *big.Int, walletid uint64, bindAddr common.Address) uint64 {

	acc, _ := self.GetAccountObject(bindAddr)
	if acc == nil {
		newacc, err := self.createAccountObject(walletid,bindAddr)
		newspa, err := self.CreateSpaceObject(id)

		if err == nil {
			self.associateSpaceAndAccount(newspa, newacc)
		}

		if newacc != nil {
			return  walletid + 1
		}
	} else {
		new, err := self.CreateSpaceObject(id)
		if err == nil {
			self.associateSpaceAndAccount(new, acc)
		}
	}

	return  walletid
}

// IntermediateAccountRoot computes the current root hash of the state trie.
func (self *StateDB) IntermediateRoot(deleteEmptyObjects bool) (rootSpace, rootAccount common.Hash) {
	rootSpace = self.IntermediateSpaceRoot(deleteEmptyObjects)
	rootAccount = self.IntermediateAccountRoot(deleteEmptyObjects)
	return rootSpace, rootAccount
}

// IntermediateAccountRoot computes the current root hash of the state trie.
func (self *StateDB) IntermediateAccountRoot(deleteEmptyObjects bool) common.Hash {
	self.AccountFinalise(deleteEmptyObjects)
	return self.trieAccount.Hash()
}

// AccountFinalise finalises the state by removing the self destructed objects
func (self *StateDB) AccountFinalise(deleteEmptyObjects bool) {
	for addr := range self.accountObjectsDirty {
		stateObject := self.accountObjects[addr]
		if deleteEmptyObjects && stateObject.empty() {
			self.deleteAccountObject(stateObject)
		} else {
			self.updateAccountObject(stateObject)
		}

		delete(self.accountObjectsDirty, addr)
	}

}

func (self *StateDB) updateAccountObject(accountObject *accountObject) {
	addr := accountObject.Address()
	data, err := rlp.EncodeToBytes(accountObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	self.setError(self.trieAccount.TryUpdate(addr[:], data))
}

// deleteStateObject removes the given object from the state trie.
func (self *StateDB) deleteAccountObject(accountObject *accountObject) {
	addr := accountObject.Address()
	self.setError(self.trieAccount.TryDelete(addr[:]))
}

// IntermediateSpaceRoot computes the current root hash of the state trie.
func (self *StateDB) IntermediateSpaceRoot(deleteEmptyObjects bool) common.Hash {
	self.SpaceFinalise(deleteEmptyObjects)
	return self.trieSpace.Hash()
}

// SpaceFinalise finalises the state by removing the self destructed objects
func (self *StateDB) SpaceFinalise(deleteEmptyObjects bool) {
	for id := range self.spaceObjectsDirty {
		stateObject := self.spaceObjects[id]
		if deleteEmptyObjects && stateObject.empty() {
			self.deleteSpaceObject(stateObject)
		} else {
			self.updateSpaceObject(stateObject)
		}

		delete(self.spaceObjectsDirty, id)
	}

}

// deleteStateObject removes the given object from the state trie.
func (self *StateDB) deleteSpaceObject(spaceObject *spaceObject) {
	id := spaceObject.Id()
	self.setError(self.trieSpace.TryDelete(id.Bytes()))
}

func (self *StateDB) updateSpaceObject(spaceObject *spaceObject) {
	id := spaceObject.Id()
	data, err := rlp.EncodeToBytes(spaceObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", id.Bytes(), err))
	}
	self.setError(self.trieSpace.TryUpdate(id.Bytes(), data))
}

func (s *StateDB) CommitTo(dbw trie.DatabaseWriter) (rootSpace, rootAccount common.Hash, err error) {

	s.SpaceFinalise(true)
	rootSpace, err = s.trieSpace.CommitTo(dbw)

	s.AccountFinalise(true)
	rootAccount, err = s.trieAccount.CommitTo(dbw)

	return rootSpace, rootAccount, err
}
