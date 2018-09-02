package blsgroup

import (
	"bytes"
	"fmt"
	"gocoin/libraries/chain/blsgroup/bls"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
	"gocoin/libraries/event"
	"gocoin/libraries/rlp"
	"strconv"
	"time"
)

type BlsGroupInfo struct {
	GID        uint
	Hash       common.Hash
	Members    []common.Address
	ID         string
	Pub        string
	Sec        string
	Gsize      uint
	Gthreshold uint
}

type CreatethresholdEvent struct {
	GID      uint
	Hash     common.Hash
	Addresss []common.Address
}

type BlsSign struct {
	ID   []byte
	Sign []byte
}

type BlsGroup struct {
	db       *lvldb.LDBDatabase
	self     common.Address
	SelfID   bls.ID
	GID      uint
	hash     common.Hash
	members  []common.Address
	k        int
	n        int
	sec      bls.SecretKey
	pub      bls.PublicKey
	blsCH    event.Feed
	secCh    chan int
	CreatCH  chan CreatethresholdEvent
	isCreate bool
}

type msec struct {
	Addr common.Address
	Msk  string
}

type BlsMsg struct {
	GID       uint
	Addr      common.Address
	Pub       string
	Times     uint
	Mastersec []msec
}
type BlsMsgEvent struct{ BlsInfo *BlsMsg }

var blsGroups map[uint]*BlsGroup
var blsHashGroups map[common.Hash]*BlsGroup
var idHash map[uint]common.Hash

func init() {
	blsGroups = make(map[uint]*BlsGroup)
	blsHashGroups = make(map[common.Hash]*BlsGroup)
	idHash = make(map[uint]common.Hash)
	bls.Init(bls.CurveFp382_2)
}

func (group *BlsGroup) CreateBlsGroup() {
	var sec bls.SecretKey
	fmt.Println("Austin ********entry CreateBlsGroup*************")
	self := group.self

	sec.SetHexString(self.Hex())
	pub := sec.GetPublicKey()

	msk := sec.GetMasterSecretKey(group.k)
	mpk := bls.GetMasterPublicKey(msk)

	idMap := make(map[common.Address]bls.ID)
	secMap := make(map[common.Address]bls.SecretKey)
	pubMap := make(map[common.Address]bls.PublicKey)

	var (
		id   bls.ID
		gsec bls.SecretKey
		gpub bls.PublicKey
	)

	for _, v := range group.members {
		id.SetLittleEndian(v[:])
		idMap[v] = id
		gsec.Set(msk, &id)
		secMap[v] = gsec
		gpub.Set(mpk, &id)
		pubMap[v] = gpub
	}
	gsec = secMap[self]
	gpub = pubMap[self]
	delete(secMap, self)
	delete(pubMap, self)

	var mastersec []msec
	for _, v := range group.members {
		hsec := secMap[v]
		hexsec := hsec.GetHexString()
		mastersec = append(mastersec, msec{Addr: v, Msk: hexsec})
	}

	msg := BlsMsg{
		GID:       group.GID,
		Addr:      self,
		Pub:       sec.GetPublicKey().GetHexString(),
		Mastersec: mastersec,
		Times:     0,
	}
	group.blsCH.Send(BlsMsgEvent{&msg})

	go func() {
		forceSync := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-group.secCh:
				var idVec []bls.ID
				var secVec []bls.SecretKey
				var pubVec []bls.PublicKey

				id.SetLittleEndian(group.self[:])
				idVec = append(idVec, id)
				secVec = append(secVec, gsec)
				pubVec = append(pubVec, *pub)
				for _, v := range psec {
					id.SetLittleEndian(v.peer[:])
					idVec = append(idVec, id)
					secVec = append(secVec, v.sec)
					pubVec = append(pubVec, v.pub)
				}
				group.sec.Recover(secVec, idVec)
				group.pub.Recover(pubVec, idVec)
				group.isCreate = true
				id.SetLittleEndian(group.self[:])
				fmt.Println("ID:", id.GetHexString())
				fmt.Println("secKey:", group.sec.GetHexString())
				fmt.Println("pubKey:", group.pub.GetHexString())
				ginfo := &BlsGroupInfo{
					GID:        group.GID,
					Hash:       group.hash,
					Members:    group.members,
					ID:         id.GetHexString(),
					Pub:        group.pub.GetHexString(),
					Sec:        group.sec.GetHexString(),
					Gsize:      uint(group.n),
					Gthreshold: uint(group.k),
				}
				err := group.saveBlsInfo(ginfo)
				if err != nil {
					fmt.Println("Austin Save bls group info")
				}
				blsGroups[group.GID] = group
			case <-forceSync.C:
				msg.Times++
				group.blsCH.Send(BlsMsgEvent{&msg})
			}
		}
	}()
}

type peerInfo struct {
	peer common.Address
	pub  bls.PublicKey
	sec  bls.SecretKey
}

var psec map[common.Address]peerInfo = make(map[common.Address]peerInfo)

func (group *BlsGroup) AddMsg(msg *BlsMsg) {
	if group.n == 0 {
		return
	}
	if _, ok := psec[msg.Addr]; ok == true {
		return
	}
	pInfo := peerInfo{}
	pInfo.peer = msg.Addr
	pInfo.pub.SetHexString(msg.Pub)

	for _, v := range msg.Mastersec {
		if v.Addr == group.self {
			pInfo.sec.SetHexString(v.Msk)
			psec[msg.Addr] = pInfo
		}
	}

	if len(psec) == group.n-1 {
		fmt.Println("Austin Send bls msg channel***************************")
		group.secCh <- 1
	}
}

func (group *BlsGroup) SubscribeRsVerifyEvent(bls chan BlsMsgEvent) {
	group.blsCH.Subscribe(bls)
}

func (group *BlsGroup) Set(self common.Address, members []common.Address, Threshold int) {
	group.self = self
	group.members = members
	group.n = len(members)
	group.SelfID.SetLittleEndian(self[:])
	group.k = Threshold
}

func (group *BlsGroup) Sign(msg []byte) *BlsSign {
	sign := group.sec.Sign(string(msg))
	blsSign := &BlsSign{
		ID:   []byte(group.SelfID.GetHexString()),
		Sign: []byte(sign.GetHexString()),
	}
	return blsSign
}

func (group *BlsGroup) Verigy(sign []BlsSign, msg []byte) (bool, []byte) {
	idVec := make([]bls.ID, len(sign))
	signVec := make([]bls.Sign, len(sign))

	var sig bls.Sign
	var id bls.ID
	for i, v := range sign {
		id.SetHexString(string(v.ID))
		idVec[i] = id
		sig.SetHexString(string(v.Sign))
		signVec[i] = sig
	}

	sig.Recover(signVec, idVec)
	ok := sig.Verify(&group.pub, string(msg))
	return ok, []byte(sig.GetHexString())
}

func (group *BlsGroup) VerigyGsign(sig []byte, msg []byte) bool {
	var sign bls.Sign
	sign.SetHexString(string(sig))
	return sign.Verify(&group.pub, string(msg))
}

func (group *BlsGroup) Size() int {
	return group.n
}

func (group *BlsGroup) Threshold() int {
	return group.k
}

func (group *BlsGroup) PubKey() []byte {
	return []byte(group.pub.GetHexString())
}

func (group *BlsGroup) Worker() {
	for {
		ev := <-group.CreatCH
		fmt.Println("Recv Create bls group:", ev.GID, ev.Hash.Hex())
		if group.isCreate == true {
			continue
		}
		ismember := false
		for _, addr := range ev.Addresss {
			fmt.Println("Austin Group addr:", addr.Hex())
			if group.self == addr {
				ismember = true
			}
			group.members = append(group.members, addr)
		}
		if ismember == false {
			group.members = group.members[:0]
			continue
		}
		if v, ok := blsHashGroups[ev.Hash]; ok == true {
			idHash[ev.GID] = ev.Hash
			blsGroups[ev.GID] = v
			group.saveIdHash(ev.GID, ev.Hash)
			continue
		}

		fmt.Println("Crete bls group")
		group.GID = ev.GID
		group.hash = ev.Hash
		group.n = len(group.members)
		group.k = group.n/2 + 1
		blsHashGroups[ev.Hash] = group
		idHash[ev.GID] = ev.Hash
		go group.CreateBlsGroup()
	}
}

func NewBlsgroup(self common.Address, members []common.Address, k int, gid uint, db lvldb.Database) *BlsGroup {
	var id bls.ID
	id.SetHexString(string(self[:]))
	group := &BlsGroup{
		db:       db.(*lvldb.LDBDatabase),
		GID:      gid,
		SelfID:   id,
		self:     self,
		members:  members,
		k:        k,
		n:        len(members),
		secCh:    make(chan int),
		CreatCH:  make(chan CreatethresholdEvent),
		isCreate: false,
	}
	group.pub.SetHexString("aaaa")
	blsGroups[gid] = group
	return group
}

func AddMsg(msg *BlsMsg) {
	if v, ok := blsGroups[msg.GID]; ok == true {
		v.AddMsg(msg)
	}
}

func GetGroup(gid uint) *BlsGroup {
	if v, ok := blsGroups[gid]; ok == true {
		return v
	}
	return nil
}

func Verify(sig, msg, pubkey []byte) bool {
	var sign bls.Sign
	var pub bls.PublicKey
	sign.SetHexString(string(sig))
	pub.SetHexString(string(pubkey))
	return sign.Verify(&pub, string(msg))
}

func (group *BlsGroup) saveIdHash(id uint, hash common.Hash) {
	gid := strconv.FormatUint(uint64(id), 10)
	group.db.Put([]byte(gid), hash[:])
}

func (group *BlsGroup) saveBlsInfo(ginfo *BlsGroupInfo) error {
	gid := strconv.FormatUint(uint64(ginfo.GID), 10)
	h := ginfo.Hash
	buf := new(bytes.Buffer)
	err := rlp.Encode(buf, ginfo)
	if err != nil {
		return err
	}
	err = group.db.Put(h[:], buf.Bytes())
	if err != nil {
		return err
	}
	err = group.db.Put([]byte(gid), h[:])
	if err != nil {
		return err
	}
	return nil
}

func (group *BlsGroup) LoadBlsInfo() error {
	fmt.Println("Load:", group.self.Hex())
	ids := []uint{0, 1, 2}
	hash := common.Hash{}
	for _, v := range ids {
		sid := strconv.FormatUint(uint64(v), 10)
		h, err := group.db.Get([]byte(sid))
		fmt.Println("Austin bls group hash:", h)
		if err == nil {
			hash = common.BytesToHash(h)
			idHash[v] = hash
			blsHashGroups[hash] = group
			blsGroups[v] = group
		}
	}

	ginfo, err := group.db.Get(hash[:])
	if err != nil {
		return err
	}
	buf := bytes.NewReader(ginfo)
	s := rlp.NewStream(buf, 0)

	var blsInfo BlsGroupInfo
	err = s.Decode(&blsInfo)
	if err != nil {
		return err
	}
	isMember := false
	for _, v := range blsInfo.Members {
		if group.self == v {
			isMember = true
			break
		}
	}

	if isMember == false {
		return fmt.Errorf("is Not member")
	}

	copy(group.members, blsInfo.Members)
	group.GID = blsInfo.GID
	group.SelfID.SetHexString(blsInfo.ID)
	group.n = int(blsInfo.Gsize)
	group.k = int(blsInfo.Gthreshold)
	group.pub.SetHexString(blsInfo.Pub)
	group.sec.SetHexString(blsInfo.Sec)
	group.isCreate = true
	fmt.Println(group)
	blsGroups[blsInfo.GID] = group
	return nil
}
