package adapter

import (
	"gocoin/libraries/common"
	"time"
)

/*
sample:
{
  "id": 11,
  "scope": "11,98",
  "location": "86",
  "recoinage": 2500,
  "allow": [
    {
      "identify": 11,
      "maxnum": 1000
    },
    {
      "identify": 12,
      "maxnum": 100
    }
  ]
}*/

type rule struct {
	Identify string
	Maxnum   uint
}

type CoinRule struct {
	ID        uint
	Scope     [2]uint
	location  string
	Recoinage uint

	Rule []rule
}

type 
 struct {
	TxTimes int
	Value   int
}

/*********************************************/
type RuleSource struct {
	K    common.Hash
	V    uint64
	VSrc []byte
}

type RuleExecute struct {
	K     common.Hash
	VTime time.Time
	VSrc  []byte
}
