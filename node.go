package main

import (
	"encoding/json"
	"net/http"
)

type Node struct {
	Blockchain   *Blockchain
	Mempool      *Mempool
	MinerAddress string
	Peers        []string
}

func NewNode(bc *Blockchain, mempool *Mempool, minerAddress string, peers []string) *Node {
	return &Node{
		Blockchain:   bc,
		Mempool:      mempool,
		MinerAddress: minerAddress,
		Peers:        append([]string(nil), peers...),
	}
}

type NodeStatus struct {
	Height      uint64 `json:"height"`
	MempoolSize int    `json:"mempool_size"`
}

func (n *Node) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if n == nil || n.Blockchain == nil || len(n.Blockchain.Blocks) == 0 || n.Mempool == nil {
		http.Error(w, "node is not initialized", http.StatusInternalServerError)
		return
	}
	tip := n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1]
	if tip == nil {
		http.Error(w, "node is not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// With this fixed schema, encoding can fail only when writing the response.
	_ = json.NewEncoder(w).Encode(NodeStatus{
		Height:      tip.Height,
		MempoolSize: n.Mempool.Size(),
	})
}

func (n *Node) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", n.StatusHandler)
	return mux
}
