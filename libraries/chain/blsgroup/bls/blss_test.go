package bls

import (
	"testing"
)

func Test_Pop(t *testing.T) {
	err := Init(CurveFp382_2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("testPop")
	var sec SecretKey
	sec.SetByCSPRNG()
	pop := sec.GetPop()
	if !pop.VerifyPop(sec.GetPublicKey()) {
		t.Errorf("Valid Pop does not verify")
	}

	sec.SetByCSPRNG()
	if pop.VerifyPop(sec.GetPublicKey()) {
		t.Errorf("Invalid Pop verifies")
	}
}

func Test_AustoGroup(t *testing.T) {
	//members := []common.Address{common.StringToAddress("1"), common.StringToAddress("2"), common.StringToAddress("3")}
	var sec1, sec2, sec3 SecretKey
	var pub PublicKey
	var id1, id2, id3 ID
	//var addr1, addr2, addr3 common.Address
	Init(CurveFp382_2)

	id1.SetHexString("69fc5ec407f238ccb681c704e6b364a4948f7da")
	id2.SetHexString("5abd1b1e2ca76a0e448b89c03a22980df64356be")
	id3.SetHexString("5a894ae266f19344d8ba92af8cd5a3ab05b1bdfd")

	sec1.SetHexString("10e1e28a44ae57de63c0869d4675c7be9af6647972fe0938fe65341c3989b23e288fe7ca1e8ce47067f3272e4a30b0e2")
	sec2.SetHexString("1ce87c710e3094c05c2fd27b664ea65c501049d9f5f795477e70feb3331121243c7d92a1f782218e5e4c92153eb790fe")
	sec3.SetHexString("22fdb39ef68925d69f7ef840a2f95e1c36cf2793436a5a950f645a5b2a57c4d9c7dba0199fd3b1692517d988adf27aa")

	pub.SetHexString("1 130b671a111a388d8a8745b0635f6d8469a79a69eab4ba1c2c6948ab181c0d2484e84611da7357f5bc4f0c8e9ee1710d 22d93f6308ed84b65f2034be0cd7fbb7ec79b3209cef49976d91561ea791d010d1af379a330b435c509a241e10063620 b358143eb5e1984a90f9e94a8e4049a7b161862d1585b1e19bc53fd858189d88afbef9ca7a3472faf93bead68061e7d e20d29ff9d708bb4b3048bd0095dc94989ccdf474d11495f60252b98a919708d189d74c3039e2702ea597b711a19790")

	msg := "Hello"
	sign1 := sec1.Sign(msg)
	t.Log(sign1.GetHexString())
	sign2 := sec2.Sign(msg)
	t.Log(sign2.GetHexString())
	sign3 := sec3.Sign(msg)
	t.Log(sign3.GetHexString())

	var sign13 Sign
	err := sign13.Recover([]Sign{*sign1, *sign3}, []ID{id1, id3})
	if err != nil {
		t.Log(err)
	}
	t.Log(sign13.Verify(&pub, msg))

	var sign12 Sign
	err = sign12.Recover([]Sign{*sign1, *sign2}, []ID{id1, id2})
	if err != nil {
		t.Log(err)
	}
	t.Log(sign12.Verify(&pub, msg))

	var sign32 Sign
	err = sign32.Recover([]Sign{*sign3, *sign2}, []ID{id3, id2})
	if err != nil {
		t.Log(err)
	}
	t.Log(sign32.Verify(&pub, msg))

	t.Log(sign1.Verify(&pub, msg))
	t.Log(sign2.Verify(&pub, msg))
	t.Log(sign3.Verify(&pub, msg))
}

func Test_Group(t *testing.T) {
	err := Init(CurveFp382_2)
	if err != nil {
		t.Fatal(err)
	}

	var sec1, sec2, sec3 SecretKey
	sec1.SetByCSPRNG()
	sec2.SetByCSPRNG()
	sec3.SetByCSPRNG()

	sec1.SetHexString("43c154378b910daeaa52b98ff5f55e68394a239d")
	sec2.SetHexString("224f49f5418e7e557d1a373092ff5a5ed6677c66")
	sec3.SetHexString("deec9148990d029f095c5c4b36189817d16d2320")

	msk1 := sec1.GetMasterSecretKey(2)
	mpk1 := GetMasterPublicKey(msk1)
	idVec1 := make([]ID, 3)
	secVec1 := make([]SecretKey, 3)
	pubVec1 := make([]PublicKey, 3)

	idVec1[0].SetHexString("9d234a39685ef5f58fb952aaae0d918b3754c143")
	secVec1[0].Set(msk1, &idVec1[0])
	pubVec1[0].Set(mpk1, &idVec1[0])

	idVec1[1].SetHexString("667c67d65e5aff9230371a7d557e8e41f5494f22")
	secVec1[1].Set(msk1, &idVec1[1])
	pubVec1[1].Set(mpk1, &idVec1[1])

	idVec1[2].SetHexString("20236dd1179818364b5c5c099f020d994891ecde")
	secVec1[2].Set(msk1, &idVec1[2])
	pubVec1[2].Set(mpk1, &idVec1[2])

	msk2 := sec2.GetMasterSecretKey(2)
	mpk2 := GetMasterPublicKey(msk2)
	idVec2 := make([]ID, 3)
	secVec2 := make([]SecretKey, 3)
	pubVec2 := make([]PublicKey, 3)

	idVec2[0].SetHexString("667c67d65e5aff9230371a7d557e8e41f5494f22")
	secVec2[0].Set(msk2, &idVec2[0])
	pubVec2[0].Set(mpk2, &idVec2[0])

	idVec2[1].SetHexString("9d234a39685ef5f58fb952aaae0d918b3754c143")
	secVec2[1].Set(msk2, &idVec2[1])
	pubVec2[1].Set(mpk2, &idVec2[1])

	idVec2[2].SetHexString("20236dd1179818364b5c5c099f020d994891ecde")
	secVec2[2].Set(msk2, &idVec2[2])
	pubVec2[2].Set(mpk2, &idVec2[2])

	msk3 := sec3.GetMasterSecretKey(2)
	mpk3 := GetMasterPublicKey(msk3)
	idVec3 := make([]ID, 3)
	secVec3 := make([]SecretKey, 3)
	pubVec3 := make([]PublicKey, 3)

	idVec3[0].SetHexString("20236dd1179818364b5c5c099f020d994891ecde")
	secVec3[0].Set(msk3, &idVec3[0])
	pubVec3[0].Set(mpk3, &idVec3[0])

	idVec3[1].SetHexString("9d234a39685ef5f58fb952aaae0d918b3754c143")
	secVec3[1].Set(msk3, &idVec3[1])
	pubVec3[1].Set(mpk3, &idVec3[1])

	idVec3[2].SetHexString("667c67d65e5aff9230371a7d557e8e41f5494f22")
	secVec3[2].Set(msk3, &idVec3[2])
	pubVec3[2].Set(mpk3, &idVec3[2])

	var gsec1 SecretKey
	var gpub1 PublicKey
	gsecVec1 := []SecretKey{secVec1[0], secVec2[1], secVec3[1]}
	gpubVec1 := []PublicKey{pubVec1[0], pubVec2[1], pubVec3[1]}
	gidVec1 := []ID{idVec1[0], idVec2[0], idVec3[0]}
	err = gsec1.Recover(gsecVec1, gidVec1)
	if err != nil {
		t.Log(err)
	}
	t.Log("gsec1:", gsec1.GetHexString())
	err = gpub1.Recover(gpubVec1, gidVec1)
	if err != nil {
		t.Log(err)
	}
	t.Log("gpub1:", gpub1.GetHexString())

	var gsec2 SecretKey
	var gpub2 PublicKey
	gsecVec2 := []SecretKey{secVec2[0], secVec1[1], secVec3[2]}
	gpubVec2 := []PublicKey{pubVec2[0], pubVec1[1], pubVec3[2]}
	gidVec2 := []ID{idVec2[0], idVec1[0], idVec3[0]}
	err = gsec2.Recover(gsecVec2, gidVec2)
	if err != nil {
		t.Log(err)
	}
	t.Log("gsec2:", gsec2.GetHexString())
	err = gpub2.Recover(gpubVec2, gidVec2)
	if err != nil {
		t.Log(err)
	}
	t.Log("gpub2:", gpub2.GetHexString())

	var gsec3 SecretKey
	var gpub3 PublicKey
	gsecVec3 := []SecretKey{secVec3[0], secVec1[2], secVec2[2]}
	gpubVec3 := []PublicKey{pubVec3[0], pubVec1[2], pubVec2[2]}
	gidVec3 := []ID{idVec3[0], idVec1[0], idVec2[0]}
	err = gsec3.Recover(gsecVec3, gidVec3)
	if err != nil {
		t.Log(err)
	}
	t.Log("gsec3:", gsec3.GetHexString())
	err = gpub3.Recover(gpubVec3, gidVec3)
	if err != nil {
		t.Log(err)
	}
	t.Log("gpub3:", gsec3.GetHexString())

	msg := "Hello"
	sign1 := gsec1.Sign(msg)
	sign3 := gsec3.Sign(msg)
	var sign Sign
	err = sign.Recover([]Sign{*sign1, *sign3}, []ID{idVec1[0], idVec3[0]})
	if err != nil {
		t.Log(err)
	}

	var gpub PublicKey
	var gsec SecretKey
	err = gpub.Recover([]PublicKey{*sec1.GetPublicKey(), *sec2.GetPublicKey(), *sec3.GetPublicKey()}, []ID{idVec1[0], idVec2[0], idVec3[0]})
	if err != nil {
		t.Log(err)
	}
	t.Log("gpubKey:", gpub.GetHexString())

	err = gsec.Recover([]SecretKey{sec1, sec2, sec3}, []ID{idVec1[0], idVec2[0], idVec3[0]})
	if err != nil {
		t.Log(err)
	}
	t.Log("gsecKey:", gsec.GetHexString())

	t.Log(sign.Verify(&gpub, msg), sign.GetHexString())
	gsign := gsec.Sign(msg)
	t.Log(gsign.Verify(&gpub, msg), gsign.GetHexString())

	var csec SecretKey
	var cpub PublicKey
	err = csec.Recover([]SecretKey{gsec1, gsec3}, []ID{idVec1[0], idVec3[0]})
	if err != nil {
		t.Log(err)
	}
	t.Log("csecKey:", csec.GetHexString())
	err = cpub.Recover([]PublicKey{gpub1, gpub3}, []ID{idVec1[0], idVec3[0]})
	if err != nil {
		t.Log(err)
	}
	t.Log("cpubKey:", cpub.GetHexString())
}
