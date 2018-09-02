/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.21
 * Website    : http:www.gocoin.com
 * Function   : space state
**/

package state

import (
	"io"
	"math/big"

	"gocoin/libraries/common"
	"gocoin/libraries/rlp"
)

type Space struct {
	Nonce    uint64
	Balance  *big.Int
	BindAddr []common.Address
}

type spaceObject struct {
	id   *big.Int
	data Space

	onDirty func(id uint64) // Callback method to mark a state object newly dirty
}

func newSpaceObject(id *big.Int, space Space, onDirty func(id uint64)) *spaceObject {
	if space.Balance == nil {
		space.Balance = new(big.Int)
	}

	return &spaceObject{
		id:      new(big.Int).Set(id),
		data:    space,
		onDirty: onDirty,
	}
}

func (c *spaceObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

func (c *spaceObject) Id() *big.Int {
	return c.id
}

func (s *spaceObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && len(s.data.BindAddr) == 0
}

func (c *spaceObject) AddBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetBalance(new(big.Int).Add(c.Balance(), amount))
}

func (c *spaceObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetBalance(new(big.Int).Sub(c.Balance(), amount))
}

func (self *spaceObject) SetBalance(amount *big.Int) {
	self.setBalance(amount)
}

func (self *spaceObject) setBalance(amount *big.Int) {
	self.data.Balance = amount
	self.data.Nonce = self.data.Nonce + 1
	if self.onDirty != nil {
		self.onDirty(self.Id().Uint64())
		self.onDirty = nil
	}
}

func (self *spaceObject) Balance() *big.Int {
	return self.data.Balance
}

func (self *spaceObject) AppendAddress(addr common.Address) {
	self.data.BindAddr = append(self.data.BindAddr, addr)

	if self.onDirty != nil {
		self.onDirty(self.Id().Uint64())
		self.onDirty = nil
	}
}

func (self *spaceObject) RemoveAddress(addr common.Address) {

	var bindAddr []common.Address
	for _, local := range self.data.BindAddr {
		if local == addr {
			continue
		} else {
			bindAddr = append(bindAddr, local)
		}
	}
	self.data.BindAddr = bindAddr

	if self.onDirty != nil {
		self.onDirty(self.Id().Uint64())
		self.onDirty = nil
	}
}
