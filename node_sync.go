package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type BlocksResponse struct {
	Blocks []Block `json:"blocks"`
}

func (n *Node) BlocksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if n == nil || n.Blockchain == nil || len(n.Blockchain.Blocks) == 0 {
		http.Error(w, "node is not initialized", http.StatusInternalServerError)
		return
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(query["from_height"]) != 1 {
		http.Error(w, "from_height is required once", http.StatusBadRequest)
		return
	}
	fromHeight, err := strconv.ParseUint(query.Get("from_height"), 10, 64)
	if err != nil {
		http.Error(w, "invalid from_height", http.StatusBadRequest)
		return
	}
	blocks := make([]Block, 0)
	for _, block := range n.Blockchain.Blocks {
		if block == nil {
			http.Error(w, "node is not initialized", http.StatusInternalServerError)
			return
		}
		if block.Height >= fromHeight {
			blocks = append(blocks, *cloneBlock(block))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(BlocksResponse{Blocks: blocks})
}

// fetchSyncJSON performs a bounded, read-only request with exactly one JSON response.
func fetchSyncJSON(peer, path string, query url.Values, destination any) error {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return fmt.Errorf("empty peer address")
	}
	endpoint, err := url.Parse(peer)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}
	if (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
		return fmt.Errorf("peer must be an HTTP(S) URL")
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	endpoint.RawPath = ""
	endpoint.RawQuery = query.Encode()
	endpoint.Fragment = ""
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Get(endpoint.String())
	if err != nil {
		return fmt.Errorf("fetch %s: %w", path, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("peer returned HTTP %d for %s", response.StatusCode, path)
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one JSON response for %s", path)
	}
	return nil
}

func (n *Node) FetchStatus(peer string) (NodeStatus, error) {
	if n == nil {
		return NodeStatus{}, fmt.Errorf("nil node")
	}
	var status *NodeStatus
	if err := fetchSyncJSON(peer, "/status", nil, &status); err != nil {
		return NodeStatus{}, err
	}
	if status == nil {
		return NodeStatus{}, fmt.Errorf("missing peer status")
	}
	return *status, nil
}

func (n *Node) FetchBlocks(peer string, fromHeight uint64) ([]Block, error) {
	if n == nil {
		return nil, fmt.Errorf("nil node")
	}
	var response *BlocksResponse
	query := url.Values{"from_height": {strconv.FormatUint(fromHeight, 10)}}
	if err := fetchSyncJSON(peer, "/blocks", query, &response); err != nil {
		return nil, err
	}
	if response == nil || response.Blocks == nil {
		return nil, fmt.Errorf("missing peer block list")
	}
	return response.Blocks, nil
}

// SyncFromPeer extends the local chain only. Accepted prefixes are not rolled back.
func (n *Node) SyncFromPeer(peer string) error {
	if n == nil || n.Blockchain == nil || n.Mempool == nil || len(n.Blockchain.Blocks) == 0 {
		return fmt.Errorf("node is not initialized")
	}
	tip := n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1]
	if tip == nil {
		return fmt.Errorf("blockchain has no tip")
	}
	status, err := n.FetchStatus(peer)
	if err != nil {
		return err
	}
	localHeight := tip.Height
	if status.Height <= localHeight {
		return nil
	}
	blocks, err := n.FetchBlocks(peer, localHeight+1)
	if err != nil {
		return err
	}
	// Check the whole batch's ordering before changing local state.
	expected := localHeight + 1
	for i := range blocks {
		if blocks[i].Height != expected {
			return fmt.Errorf("expected block height %d, got %d", expected, blocks[i].Height)
		}
		if expected == ^uint64(0) && i < len(blocks)-1 {
			return fmt.Errorf("block heights overflow uint64")
		}
		expected++
	}
	for i := range blocks {
		block := &blocks[i]
		if err := n.Blockchain.AddBlockValidated(block); err != nil {
			return fmt.Errorf("sync block %d: %w", block.Height, err)
		}
		n.Mempool.RemoveTransactions(block.Transactions)
		n.Mempool.Revalidate(n.Blockchain)
	}
	if reached := n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1].Height; reached < status.Height {
		return fmt.Errorf("incomplete sync: reached height %d, peer reported %d", reached, status.Height)
	}
	return nil
}
