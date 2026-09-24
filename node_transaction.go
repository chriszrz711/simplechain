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

type TransactionResponse struct {
	Accepted bool   `json:"accepted"`
	TxID     string `json:"txid,omitempty"`
	Error    string `json:"error,omitempty"`
}

func writeTransactionResponse(w http.ResponseWriter, status int, response TransactionResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (n *Node) TransactionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeTransactionResponse(w, http.StatusMethodNotAllowed, TransactionResponse{Error: "method not allowed"})
		return
	}
	if n == nil || n.Blockchain == nil || n.Mempool == nil {
		writeTransactionResponse(w, http.StatusInternalServerError, TransactionResponse{Error: "node is not initialized"})
		return
	}

	decoder := json.NewDecoder(r.Body)
	var tx *Transaction
	if err := decoder.Decode(&tx); err != nil || tx == nil {
		writeTransactionResponse(w, http.StatusBadRequest, TransactionResponse{Error: "invalid transaction JSON"})
		return
	}
	// Accept exactly one JSON value, allowing trailing whitespace only.
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		writeTransactionResponse(w, http.StatusBadRequest, TransactionResponse{Error: "expected one transaction"})
		return
	}
	if err := n.Mempool.AddTransaction(n.Blockchain, tx); err != nil {
		writeTransactionResponse(w, http.StatusBadRequest, TransactionResponse{Error: err.Error()})
		return
	}
	writeTransactionResponse(w, http.StatusOK, TransactionResponse{Accepted: true, TxID: fmt.Sprintf("%x", tx.ID)})
}

func (n *Node) SendTransaction(peer string, tx *Transaction) error {
	if n == nil {
		return fmt.Errorf("nil node")
	}
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return fmt.Errorf("empty peer address")
	}
	body, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("encode transaction: %w", err)
	}
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(peer, "/")+"/transaction", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create transaction request: %w", err)
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
		return fmt.Errorf("send transaction to %s: %w", peer, err)
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

func (n *Node) BroadcastTransaction(tx *Transaction) error {
	if n == nil {
		return fmt.Errorf("nil node")
	}
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}
	var failures []error
	for _, peer := range n.Peers {
		if err := n.SendTransaction(peer, tx); err != nil {
			failures = append(failures, fmt.Errorf("broadcast to %s: %w", peer, err))
		}
	}
	return errors.Join(failures...)
}

func (n *Node) SubmitTransaction(tx *Transaction) error {
	if n == nil || n.Blockchain == nil || n.Mempool == nil {
		return fmt.Errorf("node is not initialized")
	}
	if err := n.Mempool.AddTransaction(n.Blockchain, tx); err != nil {
		return err
	}
	return n.BroadcastTransaction(tx)
}
