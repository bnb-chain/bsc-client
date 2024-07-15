package blockarchiver

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
)

// JsonError represents an error in JSON format
type JsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Block represents a block in the Ethereum blockchain
type Block struct {
	WithdrawalsRoot  string        `json:"withdrawalsRoot"`
	Withdrawals      []string      `json:"withdrawals"`
	Hash             string        `json:"hash"`
	ParentHash       string        `json:"parentHash"`
	Sha3Uncles       string        `json:"sha3Uncles"`
	Miner            string        `json:"miner"`
	StateRoot        string        `json:"stateRoot"`
	TransactionsRoot string        `json:"transactionsRoot"`
	ReceiptsRoot     string        `json:"receiptsRoot"`
	LogsBloom        string        `json:"logsBloom"`
	Difficulty       string        `json:"difficulty"`
	Number           string        `json:"number"`
	GasLimit         string        `json:"gasLimit"`
	GasUsed          string        `json:"gasUsed"`
	Timestamp        string        `json:"timestamp"`
	ExtraData        string        `json:"extraData"`
	MixHash          string        `json:"mixHash"`
	Nonce            string        `json:"nonce"`
	Size             string        `json:"size"`
	TotalDifficulty  string        `json:"totalDifficulty"`
	BaseFeePerGas    string        `json:"baseFeePerGas"`
	Transactions     []Transaction `json:"transactions"`
	Uncles           []string      `json:"uncles"`
	BlobGasUsed      string        `json:"blobGasUsed"`
	ExcessBlobGas    string        `json:"excessBlobGas"`
	ParentBeaconRoot string        `json:"parentBeaconBlockRoot"`
}

// GetBlockResponse represents a response from the getBlock RPC call
type GetBlockResponse struct {
	ID      int64      `json:"id,omitempty"`
	Error   *JsonError `json:"error,omitempty"`
	Jsonrpc string     `json:"jsonrpc,omitempty"`
	Result  *Block     `json:"result,omitempty"`
}

// GetBlocksResponse represents a response from the getBlocks RPC call
type GetBlocksResponse struct {
	ID      int64      `json:"id,omitempty"`
	Error   *JsonError `json:"error,omitempty"`
	Jsonrpc string     `json:"jsonrpc,omitempty"`
	Result  []*Block   `json:"result,omitempty"`
}

// GetBundleNameResponse represents a response from the getBundleName RPC call
type GetBundleNameResponse struct {
	Data string `json:"data"`
}

// Transaction represents a transaction in the Ethereum blockchain
type Transaction struct {
	BlockHash            string        `json:"blockHash"`
	BlockNumber          string        `json:"blockNumber"`
	From                 string        `json:"from"`
	Gas                  string        `json:"gas"`
	GasPrice             string        `json:"gasPrice"`
	Hash                 string        `json:"hash"`
	Input                string        `json:"input"`
	Nonce                string        `json:"nonce"`
	To                   string        `json:"to"`
	TransactionIndex     string        `json:"transactionIndex"`
	Value                string        `json:"value"`
	Type                 string        `json:"type"`
	AccessList           []AccessTuple `json:"accessList"`
	ChainId              string        `json:"chainId"`
	V                    string        `json:"v"`
	R                    string        `json:"r"`
	S                    string        `json:"s"`
	YParity              string        `json:"yParity"`
	MaxPriorityFeePerGas string        `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         string        `json:"maxFeePerGas"`
	MaxFeePerDataGas     string        `json:"maxFeePerDataGas"`
	MaxFeePerBlobGas     string        `json:"maxFeePerBlobGas"`
	BlobVersionedHashes  []string      `json:"blobVersionedHashes"`
}

// AccessTuple represents a tuple of an address and a list of storage keys
type AccessTuple struct {
	Address     string
	StorageKeys []string
}

// GeneralBlock represents a block in the Ethereum blockchain
type GeneralBlock struct {
	*types.Block
	TotalDifficulty *big.Int `json:"totalDifficulty"` // Total difficulty in the canonical chain up to and including this block.
}

// Range represents a range of Block numbers
type Range struct {
	from uint64
	to   uint64
}

// RequestLock is a lock for making sure we don't fetch the same bundle concurrently
type RequestLock struct {
	rangeMap  map[uint64]Range
	lookupMap map[uint64]bool
	mu        sync.Mutex
}

// NewRequestLock creates a new RequestLock
func NewRequestLock() *RequestLock {
	return &RequestLock{
		rangeMap:  make(map[uint64]Range),
		lookupMap: make(map[uint64]bool),
	}
}

// IsWithinAnyRange checks if the number is within any of the cached ranges
func (rc *RequestLock) IsWithinAnyRange(num uint64) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	_, exists := rc.lookupMap[num]
	return exists
}

// AddRange adds a new range to the cache
func (rc *RequestLock) AddRange(from, to uint64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Add the range to the rangeMap
	rc.rangeMap[from] = Range{from, to}

	// Update the lookupMap for fast lookup
	for i := from; i <= to; i++ {
		rc.lookupMap[i] = true
	}
}

// RemoveRange removes a range from the cache
func (rc *RequestLock) RemoveRange(from, to uint64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Remove the range from the rangeMap
	delete(rc.rangeMap, from)

	// Update the lookupMap for fast lookup
	for i := from; i <= to; i++ {
		delete(rc.lookupMap, i)
	}
}
