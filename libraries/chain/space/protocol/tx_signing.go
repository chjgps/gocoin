/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.12.3
 * Website    : http:www.gocoin.com
 * Function   : transaction  Sign
**/

package protocol

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"gocoin/libraries/common"
	"gocoin/libraries/crypto"
)

type sigCache struct {
	signer Signer
	from   common.Address
}

func MakeSigner() Signer {
	var signer Signer
	signer = FrontierSigner{}
	return signer
}

func SignTx(tx *Tx, s Signer, prv *ecdsa.PrivateKey) (*Tx, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

func Sender(signer Signer, tx *Tx) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

type Signer interface {
	Sender(tx *Tx) (common.Address, error)
	SignatureValues(tx *Tx, sig []byte) (r, s, v *big.Int, err error)
	Hash(tx *Tx) common.Hash
	Equal(Signer) bool
}

type FrontierSigner struct{}

func (s FrontierSigner) Equal(s2 Signer) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

func (fs FrontierSigner) SignatureValues(tx *Tx, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

func (fs FrontierSigner) Hash(tx *Tx) common.Hash {
	return rlpHash([]interface{}{
		tx.data.Cost,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
	})
}

func (fs FrontierSigner) Sender(tx *Tx) (common.Address, error) {
	return recoverPlain(fs.Hash(tx), tx.data.R, tx.data.S, tx.data.V, false)
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, ErrInvalidSig
	}

	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

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
