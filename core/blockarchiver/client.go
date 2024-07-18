package blockarchiver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Client is a client to interact with the block archiver service
type Client struct {
	hc                *http.Client
	blockArchiverHost string
}

func New(blockHubHost string) (*Client, error) {
	transport := &http.Transport{
		DisableCompression:  true,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Timeout:   10 * time.Minute,
		Transport: transport,
	}
	return &Client{hc: client, blockArchiverHost: blockHubHost}, nil
}

func (c *Client) GetBlockByHash(hash common.Hash) (*Block, error) {
	payload := preparePayload("eth_getBlockByHash", []interface{}{hash.String(), "true"})
	body, err := c.postRequest(payload)
	if err != nil {
		return nil, err
	}
	getBlockResp := GetBlockResponse{}
	err = json.Unmarshal(body, &getBlockResp)
	if err != nil {
		return nil, err
	}
	return getBlockResp.Result, nil
}

func (c *Client) GetBlockByNumber(number uint64) (*Block, error) {
	payload := preparePayload("eth_getBlockByNumber", []interface{}{Int64ToHex(int64(number)), "true"})
	body, err := c.postRequest(payload)
	if err != nil {
		return nil, err
	}
	getBlockResp := GetBlockResponse{}
	err = json.Unmarshal(body, &getBlockResp)
	if err != nil {
		return nil, err
	}
	return getBlockResp.Result, nil
}

func (c *Client) GetLatestBlock() (*Block, error) {
	payload := preparePayload("eth_getBlockByNumber", []interface{}{"latest", "true"})
	body, err := c.postRequest(payload)
	if err != nil {
		return nil, err
	}
	getBlockResp := GetBlockResponse{}
	err = json.Unmarshal(body, &getBlockResp)
	if err != nil {
		return nil, err
	}
	return getBlockResp.Result, nil
}

// GetBundleBlocksRange returns the bundle blocks range
func (c *Client) GetBundleBlocksRange(blockNum uint64) (uint64, uint64, error) {
	req, err := http.NewRequest("GET", c.blockArchiverHost+fmt.Sprintf("/bsc/v1/blocks/%d/bundle/name", blockNum), nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, errors.New("failed to get bundle name")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}
	getBundleNameResp := GetBundleNameResponse{}
	err = json.Unmarshal(body, &getBundleNameResp)
	if err != nil {
		return 0, 0, err
	}
	bundleName := getBundleNameResp.Data
	parts := strings.Split(bundleName, "_")
	startSlot, err := strconv.ParseUint(parts[1][1:], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	endSlot, err := strconv.ParseUint(parts[2][1:], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return startSlot, endSlot, nil
}

// GetBundleBlocksByBlockNum returns the bundle blocks by block number that within the range
func (c *Client) GetBundleBlocksByBlockNum(blockNum uint64) ([]*Block, error) {
	payload := preparePayload("eth_getBundledBlockByNumber", []interface{}{Int64ToHex(int64(blockNum))})
	body, err := c.postRequest(payload)
	if err != nil {
		return nil, err
	}
	getBlocksResp := GetBlocksResponse{}
	err = json.Unmarshal(body, &getBlocksResp)
	if err != nil {
		return nil, err
	}
	return getBlocksResp.Result, nil
}

// postRequest sends a POST request to the block archiver service
func (c *Client) postRequest(payload map[string]interface{}) ([]byte, error) {
	// Encode payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// post call to block archiver
	req, err := http.NewRequest("POST", c.blockArchiverHost, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Perform the HTTP request
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get response")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// preparePayload prepares the payload for the request
func preparePayload(method string, params []interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
}
