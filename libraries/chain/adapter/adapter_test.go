package adapter

import (
	"testing"
)

type encRuleCase struct {
	id            int
	key           interface{}
	val           interface{}
	output, error string
}

// keywords:
// id,maxnum,scope,identify,location,recoinage
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
}
*/
var (
	ruleCases = []encRuleCase{
		{id: int(11), key: "scope", val: "none", output: ""},
		{id: int(11), key: "scope", val: "11,12", output: ""},
	}
)

func TestEncode(t *testing.T) {
}
