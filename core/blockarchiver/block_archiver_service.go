package blockarchiver

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	GetBlockRetry         = 3
	GetBlockRetryInterval = 2 * time.Second
)

var _ BlockArchiver = (*BlockArchiverService)(nil)

type BlockArchiver interface {
	GetLatestBlock() (*GeneralBlock, error)
	GetBlockByNumber(number uint64) (*types.Body, *types.Header, error)
	GetBlockByHash(hash common.Hash) (*types.Body, *types.Header, error)
}

type BlockArchiverService struct {
	// client to interact with the block archiver service
	client *Client
	// injected from BlockChain, the specified block is always read and write simultaneously in bodyCache and headerCache.
	bodyCache *lru.Cache[common.Hash, *types.Body]
	// injected from BlockChain.headerChain
	headerCache *lru.Cache[common.Hash, *types.Header]
	// hashCache is a cache for block number to hash mapping
	hashCache *lru.Cache[uint64, common.Hash]
	// requestLock is a lock to avoid concurrent fetching of the same bundle of blocks
	requestLock *RequestLock
}

// NewBlockArchiverService creates a new block archiver service
// the bodyCache and headerCache are injected from the BlockChain
func NewBlockArchiverService(blockHub string,
	bodyCache *lru.Cache[common.Hash, *types.Body],
	headerCache *lru.Cache[common.Hash, *types.Header],
	cacheSize int,
) (BlockArchiver, error) {
	client, err := New(blockHub)
	if err != nil {
		return nil, err
	}
	b := &BlockArchiverService{
		client:      client,
		bodyCache:   bodyCache,
		headerCache: headerCache,
		hashCache:   lru.NewCache[uint64, common.Hash](cacheSize),
		requestLock: NewRequestLock(),
	}
	go b.cacheStats()
	return b, nil
}

// GetLatestBlock returns the latest block
func (c *BlockArchiverService) GetLatestBlock() (*GeneralBlock, error) {
	blockResp, err := c.client.GetLatestBlock()
	if err != nil {
		return nil, err
	}
	block, err := convertBlock(blockResp)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// GetLatestHeader returns the latest header
func (c *BlockArchiverService) GetLatestHeader() (*types.Header, error) {
	block, err := c.GetLatestBlock()
	if err != nil {
		return nil, err
	}
	return block.Header(), nil
}

// GetBlockByNumber returns the block by number
func (c *BlockArchiverService) GetBlockByNumber(number uint64) (*types.Body, *types.Header, error) {
	// check if the block is in the cache
	hash, found := c.hashCache.Get(number)
	if found {
		body, foundB := c.bodyCache.Get(hash)
		header, foundH := c.headerCache.Get(hash)
		if foundB && foundH {
			return body, header, nil
		}
	}
	return c.getBlockByNumber(number)
}

// getBlockByNumber returns the block by number
func (c *BlockArchiverService) getBlockByNumber(number uint64) (*types.Body, *types.Header, error) {
	// to avoid concurrent fetching of the same bundle of blocks(), rangeCache applies here,
	// if the number is within any of the ranges, should not fetch the bundle from the block archiver service but
	// wait for a while and fetch from the cache
	if c.requestLock.IsWithinAnyRange(number) {
		// wait for a while, and fetch from the cache
		for retry := 0; retry < GetBlockRetry; retry++ {
			hash, found := c.hashCache.Get(number)
			if found {
				body, foundB := c.bodyCache.Get(hash)
				header, foundH := c.headerCache.Get(hash)
				if foundB && foundH {
					return body, header, nil
				}
			}
			time.Sleep(GetBlockRetryInterval)
		}
		// if still not found
		return nil, nil, errors.New("block not found")
	}
	// fetch the bundle range
	log.Info("fetching bundle of blocks", "number", number)
	start, end, err := c.client.GetBundleBlocksRange(number)
	if err != nil {
		return nil, nil, err
	}

	// add lock to avoid concurrent fetching of the same bundle of blocks
	c.requestLock.AddRange(start, end)
	defer c.requestLock.RemoveRange(start, end)

	blocks, err := c.client.GetBundleBlocksByBlockNum(number)
	if err != nil {
		return nil, nil, err
	}
	var body *types.Body
	var header *types.Header

	log.Info("populating block cache", "start", start, "end", end)
	for _, b := range blocks {
		block, err := convertBlock(b)
		if err != nil {
			return nil, nil, err
		}
		c.bodyCache.Add(block.Hash(), block.Body())
		c.headerCache.Add(block.Hash(), block.Header())
		c.hashCache.Add(block.NumberU64(), block.Hash())
		if block.NumberU64() == number {
			body = block.Body()
			header = block.Header()
		}
	}

	return body, header, nil
}

// GetBlockByHash returns the block by hash
func (c *BlockArchiverService) GetBlockByHash(hash common.Hash) (*types.Body, *types.Header, error) {
	body, foundB := c.bodyCache.Get(hash)
	header, foundH := c.headerCache.Get(hash)
	if foundB && foundH {
		return body, header, nil
	}

	block, err := c.client.GetBlockByHash(hash)
	if err != nil {
		return nil, nil, err
	}
	if block == nil {
		return nil, nil, nil
	}
	number, err := HexToUint64(block.Number)
	if err != nil {
		return nil, nil, err
	}
	return c.getBlockByNumber(number)
}

func (c *BlockArchiverService) cacheStats() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.Info("block archiver cache stats", "bodyCache", c.bodyCache.Len(), "headerCache", c.headerCache.Len(), "hashCache", c.hashCache.Len())
		}
	}
}
