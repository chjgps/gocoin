package publish

import (
	"crypto/ecdsa"
	"errors"
	"gocoin/libraries/common"
	"gocoin/libraries/crypto"
)

func (rs *NotaryReq) Hash() common.Hash {
	return rlpHash(rs)
}

type NotarySigner struct{}

// MakeSigner returns a Signer based on the given chain config and block number.
func MakeSigner() Signer {
	signer := NotarySigner{}
	return signer
}

func SignNotaryReq(res *NotaryReq, s Signer, prv *ecdsa.PrivateKey) (*NotaryReq, error) {
	h := s.Hash(res)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	cpy := &NotaryReq{Owner: res.Owner, Contract: res.Contract, CttHash: res.CttHash, ReqHash: res.ReqHash}
	cpy.Sig = make([]byte, 65)
	copy(cpy.Sig[:], sig)
	return cpy, nil
}

func SignNotaryRes(res *NotaryReq, s Signer, prv *ecdsa.PrivateKey) ([]byte, error) {
	h := s.Hash(res)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func Sender(signer Signer, res *NotaryReq) (common.Address, error) {
	addr, err := signer.Sender(res)
	if err != nil {
		return common.Address{}, err
	}
	return addr, nil
}

type Signer interface {

	// Sender returns the sender address of the transaction.
	Sender(res *NotaryReq) (common.Address, error)
	// Hash returns the hash to be signed.
	Hash(res *NotaryReq) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

func (s NotarySigner) Equal(s2 Signer) bool {
	_, ok := s2.(NotarySigner)
	return ok
}

func (fs NotarySigner) Hash(res *NotaryReq) common.Hash {
	return rlpHash([]interface{}{
		res.Owner,
		res.Contract,
		res.CttHash,
		res.ReqHash,
	})
}

func (fs NotarySigner) Sender(res *NotaryReq) (common.Address, error) {
	return recoverPlain(fs.Hash(res), res.Sig)
}

func recoverPlain(sighash common.Hash, sig []byte) (common.Address, error) {
	if len(sig) != 65 {
		return common.Address{}, errors.New("wrong size for signature ")
	}

	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}
