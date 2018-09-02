/**
 *
 * Copyright  : (C) 2018 gocoin Team
 * LastModify : 2017.11.23
 * Website    : http:www.gocoin.com
 * Function   : block chain header
**/

package space

import (
	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/common"
	"gocoin/libraries/db/lvldb"
)

type HeaderChain struct {
	chainDb       lvldb.Database
	genesisHeader *protocol.Header

	currentHeader     *protocol.Header
	currentHeaderHash common.Hash
}

func NewHeaderChain(chainDb lvldb.Database) (*HeaderChain, error) {

	hc := &HeaderChain{
		chainDb: chainDb,
	}

	hc.genesisHeader = hc.GetHeaderByNumber(0)

	hc.currentHeader = hc.genesisHeader
	if head := GetHeadBlockHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			hc.currentHeader = chead
		}
	}
	hc.currentHeaderHash = hc.currentHeader.Hash()

	return hc, nil
}

func (hc *HeaderChain) GetBlockNumber(hash common.Hash) uint64 {
	number := GetBlockNumber(hc.chainDb, hash)

	return number
}

func (hc *HeaderChain) WriteHeader(header *protocol.Header) error {

	//hash := header.Hash()
	if err := WriteHeader(hc.chainDb, header); err != nil {
		return err
	}

	return nil
}

func (hc *HeaderChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	header := hc.GetHeaderByHash(hash)
	if header == nil {
		return nil
	}

	chain := make([]common.Hash, 0, max)
	for i := uint64(0); i < max; i++ {
		next := header.UncleHash
		if header = hc.GetHeader(next, header.Number.Uint64()-1); header == nil {
			break
		}
		chain = append(chain, next)
		if header.Number.Sign() == 0 {
			break
		}
	}
	return chain

}

func (hc *HeaderChain) GetHeaderByHash(hash common.Hash) *protocol.Header {
	return hc.GetHeader(hash, hc.GetBlockNumber(hash))
}

func (hc *HeaderChain) HasHeader(hash common.Hash, number uint64) bool {
	ok, _ := hc.chainDb.Has(headerKey(hash, number))
	return ok
}

func (hc *HeaderChain) GetHeader(hash common.Hash, number uint64) *protocol.Header {
	header := GetHeader(hc.chainDb, hash, number)
	if header == nil {
		return nil
	}
	return header
}

func (hc *HeaderChain) GetHeaderByNumber(number uint64) *protocol.Header {
	hash := GetCanonicalHash(hc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash, number)
}

func (hc *HeaderChain) CurrentHeader() *protocol.Header {
	return hc.currentHeader
}

func (hc *HeaderChain) SetCurrentHeader(head *protocol.Header) {
	if err := WriteHeadHeaderHash(hc.chainDb, head.Hash()); err == nil {
		hc.currentHeader = head
		hc.currentHeaderHash = head.Hash()
	}
}

func (hc *HeaderChain) SetGenesis(head *protocol.Header) {
	hc.genesisHeader = head
}
