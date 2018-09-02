/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : genesis  block
**/
package space

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"

	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/chain/space/state"
	"gocoin/libraries/chain/validategroup"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
)

var errGenesisNoConfig = errors.New("genesis has no chain configuration")
var genesisVersion = []byte("20180105")

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	//Config     *params.ChainConfig `json:"config"`
	ChainId    *big.Int    `json:"chainId"`
	Number     int64       `json:"number"`
	ParentHash common.Hash `json:"parentHash"`
	Timestamp  int64       `json:"timestamp"`
	Version    []byte      `json:"Version"`
	Root       common.Hash `json:"stateRoot"        gencodec:"required"`
	TxHash     common.Hash `json:"transactionsRoot" gencodec:"required"`
}

type GenesisConfig struct {
	XMLName  xml.Name       `xml:"Config"`
	GAccount GenesisAccount `xml:"Genesis"`
}

type GenesisAccount struct {
	Account []AccountConfig `xml:"Account"`
}

type AccountConfig struct {
	Address string    `xml:"Address,attr"`
	SpaceId uint64    `xml:"SpaceId,attr"`
	Balance uint64    `xml:"Balance,attr"`
	WalletId uint64   `xml:"WalletId,attr"`
	GroupId  uint32   `xml:"GroupId,attr"`
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func Parser(FileName string) GenesisConfig {
	file, err := os.Open(FileName)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	root := GenesisConfig{}
	err = xml.Unmarshal(data, &root)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(root.XMLName)
	for _, ac := range root.GAccount.Account {
	   fmt.Println("Address : ", ac.Address)
	    fmt.Println("SpaceId : ", ac.SpaceId)
	    fmt.Println("Balance : ", ac.Balance)
		fmt.Println("WalletId : ", ac.WalletId)
		fmt.Println("GroupId : ", ac.GroupId)
	}

	return root
}
func MakeXml(XmlFile string, root GenesisConfig) {
	f, err := os.Create(XmlFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	f.Write([]byte(xml.Header))
	output, err := xml.MarshalIndent(root, "  ", "    ")
	f.Write(output)
	return
}

// The returned chain configuration is never nil.
func GetGenesisBlock(db lvldb.Database, dataDir string) (*protocol.Block, error) {

	// Check whether the genesis block is already written.
	hash := GetHeadBlockHash(db)
	fmt.Println("GetHeadBlockHash  ", hash)
	if (hash == common.Hash{}) {

		block, address := ToBlock(db, dataDir)
		if block == nil {
			return nil, errGenesisNoConfig
		} else {
			if err := WriteBlock(db, block); err != nil {
				return nil, err
			}

			if err := WriteCanonicalHash(db, block.Hash(), block.GetNumberU64()); err != nil {
				return nil, err
			}
			if err := WriteHeadBlockHash(db, block.Hash()); err != nil {
				return nil, err
			}
			if err := WriteHeadHeaderHash(db, block.Hash()); err != nil {
				return nil, err
			}

			validataGroup := validategroup.NewValidateGroup(0, block.GetConsensusGroupHash(), db)
			groupuser := validategroup.GroupMember{
				Address: address,
				Time:    big.NewInt(time.Now().UnixNano()),
			}

			validataGroup.Add(groupuser)

			validataGroup.Reset(protocol.RlpHash(address), true)
			size := validataGroup.GroupSize()
			fmt.Println("validataGroup size : ", size)
			return block, nil
		}
	}

	return nil, errGenesisNoConfig
}

// ToBlock creates the block and state of a genesis specification.
func ToBlock(db lvldb.Database, dataDir string) (*protocol.Block, common.Address) {

	state, _ := state.New(common.Hash{}, common.Hash{}, state.NewDatabase(db))

	filename := filepath.Join(dataDir, "confing.xml")
	flag, _ := PathExists(filename)

	if flag {
		root := Parser(filename)
		var addre common.Address
		var walletindex  uint64
		var  groupid  uint32

		for index, ac := range root.GAccount.Account {
			addre1 := common.HexToAddress(ac.Address)
			fmt.Println("genesis Addr : ", addre1.Hex(), "   ", addre1)
			walletindex = state.CreateSpaceAndBindAddr(big.NewInt(int64(ac.SpaceId)),ac.WalletId, addre1)
			state.SetBalance(addre1, big.NewInt(int64(ac.Balance)))
			if index == 0 {
				addre = addre1
				groupid = ac.GroupId
			}
		}

		fmt.Println("walletidex : ", walletindex )
		rootSpace, ParentHash, _ := state.CommitTo(db)
		groupConstHash := protocol.RlpHash(addre)
		t := time.Date(2018, time.January, 1, 8, 0, 0, 0, time.UTC)
		head := &protocol.Header{
			Number:         big.NewInt(0),
			Timestamp:      big.NewInt(t.Unix()),
			UncleHash:      common.Hash{},
			Version:        genesisVersion,
			IdIndex:        big.NewInt(1),
			WalletID:       walletindex,
			GroupID:        groupid,
			TxHash:         common.Hash{},
			SpaceRoot:      rootSpace,
			AccRoot:        ParentHash,
			CoinRoot:       common.Hash{},
			GroupRoot:      common.Hash{},
			ConsensusGroupHash: groupConstHash,
		}

		return protocol.NewBlock(head, &protocol.Block_Body{}), addre

	} else {

		fmt.Println(" config.xml  file does not exist ")
		return nil, common.Address{}
	}

}
