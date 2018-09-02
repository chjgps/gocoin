package blsgroup

import (
	"gocoin/libraries/chain/blsgroup/bls"
	"gocoin/libraries/db/lvldb"
	"testing"
)

func (group *BlsGroup) SetKey(sec bls.SecretKey, pub bls.PublicKey, id bls.ID) {
	group.sec = sec
	group.pub = pub
	group.SelfID = id
}

func Test_blsSig(t *testing.T) {
	bls.Init(bls.CurveFp382_2)

	var sec1, sec2, sec3 bls.SecretKey
	var id1, id2, id3 bls.ID
	var pub bls.PublicKey

	id1.SetHexString("69fc5ec407f238ccb681c704e6b364a4948f7da")
	id2.SetHexString("5abd1b1e2ca76a0e448b89c03a22980df64356be")
	id3.SetHexString("5a894ae266f19344d8ba92af8cd5a3ab05b1bdfd")

	sec1.SetHexString("22c6f8777f58d95abb1b276e0ca94a49aab340152842308ecd03e8b0abad20b7a8abb18979dbb03b072b769b7542f750")
	sec2.SetHexString("1f057f22aa70abcd41e482dd0a6a6f3d332e000bdd90c00eff18dc8063fa0d409ddb2755e1f2298a001f852077241031")
	sec3.SetHexString("1cb11a6b18721abec133658b4feba7cf9fe17e987286f3c57cc73e1dedc709bd3e0afd93beaee2ffe811e9f52aa6a469")

	pub.SetHexString("1 130b671a111a388d8a8745b0635f6d8469a79a69eab4ba1c2c6948ab181c0d2484e84611da7357f5bc4f0c8e9ee1710d 22d93f6308ed84b65f2034be0cd7fbb7ec79b3209cef49976d91561ea791d010d1af379a330b435c509a241e10063620 b358143eb5e1984a90f9e94a8e4049a7b161862d1585b1e19bc53fd858189d88afbef9ca7a3472faf93bead68061e7d e20d29ff9d708bb4b3048bd0095dc94989ccdf474d11495f60252b98a919708d189d74c3039e2702ea597b711a19790")

	blsGroup1 := new(BlsGroup)
	blsGroup1.SetKey(sec1, pub, id1)

	blsGroup2 := new(BlsGroup)
	blsGroup2.SetKey(sec2, pub, id2)

	blsGroup3 := new(BlsGroup)
	blsGroup3.SetKey(sec3, pub, id3)

	msg := []byte("Hello")
	msg1 := []byte("Hell")
	sign1 := blsGroup1.Sign(msg)
	sign2 := blsGroup2.Sign(msg)
	sign3 := blsGroup3.Sign(msg)

	t.Log(blsGroup1.Verigy([]BlsSign{*sign1, *sign2}, msg))
	t.Log(blsGroup1.Verigy([]BlsSign{*sign3, *sign2}, msg))
	t.Log(blsGroup1.Verigy([]BlsSign{*sign3, *sign1}, msg))
	t.Log(blsGroup1.Verigy([]BlsSign{*sign3, *sign1, *sign2}, msg))

	t.Log(blsGroup1.Verigy([]BlsSign{*sign2}, msg))
	t.Log(blsGroup1.Verigy([]BlsSign{*sign3}, msg))
	t.Log(blsGroup1.Verigy([]BlsSign{*sign3}, msg))

	sign2 = blsGroup2.Sign(msg1)
	t.Log(blsGroup1.Verigy([]BlsSign{*sign3, *sign1, *sign2}, msg))
}

func Test_empty(t *testing.T) {
	bls.Init(bls.CurveFp382_2)
	blsGroup := new(BlsGroup)
	var bsign BlsSign
	bsign.ID = []byte{0}
	bsign.Sign = []byte{0}
	blsGroup.pub.SetHexString("Hello")
	ok, _ := blsGroup.Verigy([]BlsSign{bsign}, []byte("hello666666"))
	t.Log(ok)
}

func Test_BlsGroupInfo(t *testing.T) {
	db, _ := lvldb.NewLDBDatabase("/tmp/data", 0, 0)
	_ = db
}
