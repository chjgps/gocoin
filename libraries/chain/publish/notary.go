package publish

import (
	"gocoin/libraries/chain/transaction/gwallet"
	"gocoin/libraries/common"
	"gocoin/libraries/event"
)

const MAX_NOTARY = 3

type NotaryReq struct {
	Owner    common.Address
	Contract []byte
	CttHash  common.Hash
	Sig      []byte
	ReqHash  common.Hash
	Notary   []Scrivener
}

type NotaryRes NotaryReq

type NotaryReqEvent struct {
	Msg *NotaryReq
}

type Notary struct {
	notaryCh  event.Feed
	recvCh    map[common.Hash]chan *NotaryRes
	contracts map[common.Hash]bool
	notary    map[common.Hash][]Scrivener
}

func (nt *Notary) SubscribeNotaryEvent(event chan NotaryReqEvent) {
	nt.notaryCh.Subscribe(event)
}

func (nt *Notary) RecvNotaryReq(msg *NotaryReq) (*NotaryRes, bool) {
	if len(msg.Notary) > MAX_NOTARY {
		return nil, false
	}

	signer := MakeSigner()
	sig, err := SignNotaryRes(msg, signer, gwallet.GetGWallet(0).Gprv)
	if err != nil {
		return nil, false
	}

	res := (*NotaryRes)(msg)
	res.Notary = append(res.Notary, Scrivener{Addr: gwallet.GetGWallet(0).Gaddr, Sign: sig})
	return res, true
}

func (nt *Notary) RecvNotaryRes(msg *NotaryRes) {
	if nt.contracts[msg.CttHash] == false {
		return
	}
	nt.notary[msg.CttHash] = append(nt.notary[msg.CttHash], msg.Notary...)
	nt.recvCh[msg.CttHash] <- msg
}

func (nt *Notary) WaitNotary(hash common.Hash) *NotaryRes {
	var msg *NotaryRes
	for i := 0; i < MAX_NOTARY-1; i++ {
		msg = <-nt.recvCh[hash]
	}
	msg.Notary = nt.notary[hash]
	return msg
}

func (nt *Notary) ReqNotary(address common.Address, contract []byte) *NotaryReq {
	h := rlpHash(contract)
	req := &NotaryReq{
		Owner:    address,
		Contract: contract,
		CttHash:  h,
	}
	notary.contracts[h] = true
	notary.recvCh[h] = make(chan *NotaryRes)
	nt.notaryCh.Send(NotaryReqEvent{req})
	return req
}

var notary *Notary = nil

func GetNotary() *Notary {
	if notary != nil {
		return notary
	}
	notary = &Notary{
		recvCh:    make(map[common.Hash]chan *NotaryRes),
		contracts: make(map[common.Hash]bool),
		notary:    make(map[common.Hash][]Scrivener),
	}
	return notary
}
