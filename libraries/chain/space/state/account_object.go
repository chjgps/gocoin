/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.21
 * Website    : http:www.gocoin.com
 * Function   : Account state
**/

package state

import (
	"io"
	"math/big"

	"gocoin/libraries/common"
	"gocoin/libraries/rlp"
)

type Code []byte

type Account struct {
	BindSpaceId  *big.Int
	Balance      *big.Int
	WalletID     uint64
	CodeHash     []byte
	Accountkey   []string
	AccountValue []string
}

type accountObject struct {
	address  common.Address
	addrHash common.Hash

	data Account
	code Code

	accountAttribute map[string]string
	onDirty          func(addr common.Address) // Callback method to mark a state object newly dirty
}

func newAccountObject(address common.Address, acc Account, onDirty func(addr common.Address)) *accountObject {
	if acc.BindSpaceId == nil {
		acc.BindSpaceId = new(big.Int)
	}
	if acc.Balance == nil {
		acc.Balance = new(big.Int)
	}

	newaccount := &accountObject{
		address:          address,
		data:             acc,
		onDirty:          onDirty,
		accountAttribute: make(map[string]string),
	}

	size := len(newaccount.data.Accountkey)

	if size > len(newaccount.data.AccountValue) {
		size = len(newaccount.data.AccountValue)
	}

	for i := 0; i < size; i++ {
		newaccount.accountAttribute[newaccount.data.Accountkey[i]] = newaccount.data.AccountValue[i]
	}

	return newaccount
}

func (self *accountObject) EncodeRLP(w io.Writer) error {

	self.data.Accountkey = make([]string, len(self.accountAttribute))
	self.data.AccountValue = make([]string, len(self.accountAttribute))

	index := 0
	for key, value := range self.accountAttribute {
		self.data.Accountkey[index] = key
		self.data.AccountValue[index] = value
		index++
	}

	return rlp.Encode(w, self.data)
}

func (self *accountObject) Address() common.Address {
	return self.address
}

func (self *accountObject) empty() bool {
	return self.data.BindSpaceId.Sign() == 0 && len(self.accountAttribute) == 0 && self.data.Balance.Sign() == 0
}

func (c *accountObject) SpaceID() *big.Int {
	return c.data.BindSpaceId
}

func (c *accountObject) GetBalance() *big.Int {
	return c.data.Balance
}

func (c *accountObject) GetWalletID() uint64 {
	return c.data.WalletID
}

func (c *accountObject) GetCodeHash() []byte {
	return c.data.CodeHash
}

func (self *accountObject) AddBalance_gas(amount *big.Int) {

	if amount.Sign() == 0 {
		if self.empty() {
			return
		}
		return
	}
	self.SetBalance_gas(new(big.Int).Add(self.GetBalance(), amount))
}

// SubBalance removes amount from c's balance.
// It is used to remove funds from the origin account of a transfer.
func (self *accountObject) SubBalance_gas(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	self.SetBalance_gas(new(big.Int).Sub(self.GetBalance(), amount))
}

func (self *accountObject) SetBalance_gas(amount *big.Int) {
	self.setBalance_gas(amount)
}

func (self *accountObject) setBalance_gas(amount *big.Int) {
	self.data.Balance = amount

	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *accountObject) SetSpaceID(id *big.Int) {
	self.data.BindSpaceId = new(big.Int).Set(id)

	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *accountObject) GetAttribute(key []byte) []byte {

	strkey := string(key[:])
	if str, ok := self.accountAttribute[strkey]; ok {
		return []byte(str)
	}

	return []byte{}
}

func (self *accountObject) SetAttribute(key, value []byte) {

	strkey := string(key[:])
	strvalue := string(value[:])
	self.accountAttribute[strkey] = strvalue

	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *accountObject) DelAttribute(key []byte) {

	strkey := string(key[:])
	delete(self.accountAttribute, strkey)

	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}
