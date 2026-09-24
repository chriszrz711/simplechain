package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type BlockResponse struct {
	Accepted bool   `json:"accepted"`
	Height   uint64 `json:"height,omitempty"`
	Hash     string `json:"hash,omitempty"`
	Error    string `json:"error,omitempty"`
}

func writeBlockResponse(w http.ResponseWriter, status int, response BlockResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (n *Node) BlockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeBlockResponse(w, http.StatusMethodNotAllowed, BlockResponse{Error: "method not allowed"})
		return
	}
	if n == nil || n.Blockchain == nil || n.Mempool == nil {
		writeBlockResponse(w, http.StatusInternalServerError, BlockResponse{Error: "node is not initialized"})
		return
	}
	decoder := json.NewDecoder(r.Body)
	var block *Block
	if err := decoder.Decode(&block); err != nil || block == nil {
		writeBlockResponse(w, http.StatusBadRequest, BlockResponse{Error: "invalid block JSON"})
		return
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		writeBlockResponse(w, http.StatusBadRequest, BlockResponse{Error: "expected one block"})
		return
	}

	if len(n.Blockchain.Blocks) > 0 {
		tip := n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1]
		if tip == nil {
			writeBlockResponse(w, http.StatusInternalServerError, BlockResponse{Error: "node is not initialized"})
			return
		}
		if block.Height == tip.Height && bytes.Equal(block.Hash, tip.Hash) {
			writeBlockResponse(w, http.StatusOK, BlockResponse{Accepted: true, Height: tip.Height, Hash: fmt.Sprintf("%x", tip.Hash)})
			return
		}
	}
	if err := n.Blockchain.AddBlockValidated(block); err != nil {
		writeBlockResponse(w, http.StatusBadRequest, BlockResponse{Error: err.Error()})
		return
	}
	n.Mempool.RemoveTransactions(block.Transactions)
	n.Mempool.Revalidate(n.Blockchain)
	writeBlockResponse(w, http.StatusOK, BlockResponse{Accepted: true, Height: block.Height, Hash: fmt.Sprintf("%x", block.Hash)})
}

func (n *Node) SendBlock(peer string, block *Block) error {
	if n == nil {
		return fmt.Errorf("nil node")
	}
	if block == nil {
		return fmt.Errorf("nil block")
	}
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return fmt.Errorf("empty peer address")
	}
	body, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("encode block: %w", err)
	}
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(peer, "/")+"/block", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create block request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send block to %s: %w", peer, err)
	}
	defer response.Body.Close()
	_, readErr := io.Copy(io.Discard, response.Body)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("peer %s returned HTTP %d", peer, response.StatusCode)
	}
	if readErr != nil {
		return fmt.Errorf("read peer response: %w", readErr)
	}
	return nil
}

func (n *Node) BroadcastBlock(block *Block) error {
	if n == nil {
		return fmt.Errorf("nil node")
	}
	if block == nil {
		return fmt.Errorf("nil block")
	}
	var failures []error
	for _, peer := range n.Peers {
		if err := n.SendBlock(peer, block); err != nil {
			failures = append(failures, fmt.Errorf("broadcast to %s: %w", peer, err))
		}
	}
	return errors.Join(failures...)
}

// MineAndBroadcastBlock commits local mining before attempting peer delivery.
// A broadcast failure returns the accepted block alongside the error.
func (n *Node) MineAndBroadcastBlock() (*Block, error) {
	if n == nil || n.Blockchain == nil || n.Mempool == nil {
		return nil, fmt.Errorf("node is not initialized")
	}
	if n.MinerAddress == "" {
		return nil, fmt.Errorf("miner address cannot be empty")
	}
	block, err := n.Blockchain.MineBlock(n.Mempool, n.MinerAddress)
	if err != nil {
		return nil, err
	}
	return block, n.BroadcastBlock(block)
}
