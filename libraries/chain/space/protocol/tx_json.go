/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.12.3
 * Website    : http:www.gocoin.com
 * Function   : transaction json Encode  and  Decode
**/

package protocol

import (
	"encoding/json"
	"errors"
	"math/big"

	"gocoin/libraries/common"
	"gocoin/libraries/common/hexutil"
)

func (t *Txstruct) MarshalJSON() ([]byte, error) {
	type txstruct struct {
		Cost      *hexutil.Big    `json:"cost"      gencodec:"required"`
		Recipient *common.Address `json:"to"       rlp:"nil"`
		Amount    *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload   hexutil.Bytes   `json:"input"    gencodec:"required"`
		V         *hexutil.Big    `json:"v" gencodec:"required"`
		R         *hexutil.Big    `json:"r" gencodec:"required"`
		S         *hexutil.Big    `json:"s" gencodec:"required"`
		Hash      *common.Hash    `json:"hash" rlp:"-"`
	}
	var enc txstruct
	enc.Cost = (*hexutil.Big)(t.Cost)
	enc.Recipient = t.Recipient
	enc.Amount = (*hexutil.Big)(t.Amount)
	enc.Payload = t.Payload
	enc.V = (*hexutil.Big)(t.V)
	enc.R = (*hexutil.Big)(t.R)
	enc.S = (*hexutil.Big)(t.S)
	enc.Hash = t.Hash
	return json.Marshal(&enc)
}

func (t *Txstruct) UnmarshalJSON(input []byte) error {
	type txstruct struct {
		Cost      *hexutil.Big    `json:"cost"      gencodec:"required"`
		Recipient *common.Address `json:"to"       rlp:"nil"`
		Amount    *hexutil.Big    `json:"value"    gencodec:"required"`
		Payload   hexutil.Bytes   `json:"input"    gencodec:"required"`
		V         *hexutil.Big    `json:"v" gencodec:"required"`
		R         *hexutil.Big    `json:"r" gencodec:"required"`
		S         *hexutil.Big    `json:"s" gencodec:"required"`
		Hash      *common.Hash    `json:"hash" rlp:"-"`
	}
	var dec txstruct
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Cost == nil {
		return errors.New("missing required field 'gas' for txstruct")
	}
	t.Cost = (*big.Int)(dec.Cost)
	if dec.Recipient != nil {
		t.Recipient = dec.Recipient
	}
	if dec.Amount == nil {
		return errors.New("missing required field 'value' for txstruct")
	}
	t.Amount = (*big.Int)(dec.Amount)
	if dec.Payload == nil {
		return errors.New("missing required field 'input' for txstruct")
	}
	t.Payload = dec.Payload
	if dec.V == nil {
		return errors.New("missing required field 'v' for txstruct")
	}
	t.V = (*big.Int)(dec.V)
	if dec.R == nil {
		return errors.New("missing required field 'r' for txstruct")
	}
	t.R = (*big.Int)(dec.R)
	if dec.S == nil {
		return errors.New("missing required field 's' for txstruct")
	}
	t.S = (*big.Int)(dec.S)
	if dec.Hash != nil {
		t.Hash = dec.Hash
	}
	return nil
}
