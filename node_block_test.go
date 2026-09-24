package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func blockNodes(t *testing.T, count int) ([]*Node, *Transaction, *Wallet) {
	t.Helper()
	alice, bob := NewWallet(), NewWallet()
	bc := NewBlockchain()
	if err := bc.AddBlock([]Transaction{*NewCoinbaseTransaction(alice.Address(), CoinbaseReward)}); err != nil {
		t.Fatal(err)
	}
	nodes := []*Node{NewNode(bc, NewMempool(), NewWallet().Address(), nil)}
	// Replay the exact same mined parent through safe admission on every peer.
	body, _ := json.Marshal(bc.Blocks[1])
	for i := 1; i < count; i++ {
		peer := NewBlockchain()
		var funding Block
		if err := json.Unmarshal(body, &funding); err != nil {
			t.Fatal(err)
		}
		if err := peer.AddBlockValidated(&funding); err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, NewNode(peer, NewMempool(), NewWallet().Address(), nil))
	}
	tx, err := bc.NewSignedTransaction(alice, bob.Address(), 30)
	if err != nil {
		t.Fatal(err)
	}
	return nodes, tx, alice
}
func postBlock(t *testing.T, n *Node, body []byte) (int, BlockResponse) {
	t.Helper()
	recorder := httptest.NewRecorder()
	n.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/block", bytes.NewReader(body)))
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected JSON response")
	}
	var response BlockResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return recorder.Code, response
}
func blockNodeSnapshot(t *testing.T, n *Node) []byte {
	t.Helper()
	body, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func TestBlockHandlerAcceptsAndDeduplicates(t *testing.T) {
	nodes, tx, _ := blockNodes(t, 2)
	for _, n := range nodes {
		if err := n.Mempool.AddTransaction(n.Blockchain, tx); err != nil {
			t.Fatal(err)
		}
	}
	var calls atomic.Int32
	observer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer observer.Close()
	nodes[1].Peers = []string{observer.URL}
	block, err := nodes[0].Blockchain.MineBlock(nodes[0].Mempool, nodes[0].MinerAddress)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(block)
	length := len(nodes[1].Blockchain.Blocks)
	for i := 0; i < 2; i++ {
		code, response := postBlock(t, nodes[1], body)
		if code != 200 || !response.Accepted || response.Height != block.Height || response.Hash != fmt.Sprintf("%x", block.Hash) || response.Error != "" {
			t.Fatalf("unexpected response: %d %+v", code, response)
		}
		if len(nodes[1].Blockchain.Blocks) != length+1 {
			t.Fatal("duplicate block must not append twice")
		}
	}
	peer := nodes[1]
	if !bytes.Equal(peer.Blockchain.Blocks[length].Hash, block.Hash) || !peer.Blockchain.ValidateChain() {
		t.Fatal("peer must accept the valid mined tip")
	}
	if peer.Mempool.Size() != 0 || peer.Mempool.Has(tx.ID) {
		t.Fatal("confirmed transaction must leave receiver mempool")
	}
	if calls.Load() != 0 {
		t.Fatal("received blocks must not be rebroadcast")
	}
}
func TestBlockHandlerRejectsInvalidBlocksWithoutMutation(t *testing.T) {
	nodes, tx, _ := blockNodes(t, 2)
	receiver := nodes[1]
	if err := receiver.Mempool.AddTransaction(receiver.Blockchain, tx); err != nil {
		t.Fatal(err)
	}
	block, err := nodes[0].Blockchain.MineBlock(nodes[0].Mempool, nodes[0].MinerAddress)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"wrong_height", "wrong_prev_hash", "invalid_pow", "invalid_transaction"} {
		t.Run(name, func(t *testing.T) {
			body, _ := json.Marshal(block)
			var bad Block
			if err := json.Unmarshal(body, &bad); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "wrong_height":
				bad.Height++
			case "wrong_prev_hash":
				bad.PrevHash = []byte("wrong")
			case "invalid_pow":
				bad.Hash = []byte("wrong")
			case "invalid_transaction":
				bad.Transactions[0].Amount++
				bad.Transactions[0].SetID()
			}
			if name != "invalid_pow" {
				bad.Nonce, bad.Hash = NewProofOfWork(&bad).Run()
			}
			before := blockNodeSnapshot(t, receiver)
			body, _ = json.Marshal(bad)
			code, response := postBlock(t, receiver, body)
			if code != 400 || response.Accepted || response.Error == "" {
				t.Fatalf("expected rejection: %d %+v", code, response)
			}
			if !bytes.Equal(before, blockNodeSnapshot(t, receiver)) {
				t.Fatal("failed admission must preserve blockchain, UTXOs and mempool")
			}
		})
	}
	// Mine a different valid sibling on the receiver, then reject the sender's block.
	if _, err := receiver.Blockchain.MineBlock(receiver.Mempool, receiver.MinerAddress); err != nil {
		t.Fatal(err)
	}
	before := blockNodeSnapshot(t, receiver)
	body, _ := json.Marshal(block)
	if code, _ := postBlock(t, receiver, body); code != 400 {
		t.Fatal("different block at same height must be rejected")
	}
	if !bytes.Equal(before, blockNodeSnapshot(t, receiver)) {
		t.Fatal("fork rejection must preserve state")
	}
}
func TestBlockHandlerRejectsBadRequests(t *testing.T) {
	nodes, _, _ := blockNodes(t, 1)
	n := nodes[0]
	before := blockNodeSnapshot(t, n)
	for _, body := range [][]byte{nil, []byte("null"), []byte("{"), []byte("{} {}"), []byte("{} trailing")} {
		if code, response := postBlock(t, n, body); code != 400 || response.Accepted {
			t.Fatal("malformed body must be rejected")
		}
	}
	recorder := httptest.NewRecorder()
	n.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/block", nil))
	if recorder.Code != 405 || recorder.Header().Get("Allow") != "POST" {
		t.Fatal("GET must return 405 with Allow POST")
	}
	if !bytes.Equal(before, blockNodeSnapshot(t, n)) {
		t.Fatal("bad requests must not mutate state")
	}
	for _, bad := range []*Node{nil, NewNode(nil, NewMempool(), "", nil), NewNode(NewBlockchain(), nil, "", nil)} {
		if code, _ := postBlock(t, bad, []byte("{}")); code != 500 {
			t.Fatal("uninitialized nodes must return 500")
		}
	}
}
func TestBlockHandlerRevalidatesConflictingPending(t *testing.T) {
	nodes, pending, alice := blockNodes(t, 2)
	if err := nodes[1].Mempool.AddTransaction(nodes[1].Blockchain, pending); err != nil {
		t.Fatal(err)
	}
	external, err := nodes[0].Blockchain.NewSignedTransaction(alice, "Charlie", 12)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodes[0].Mempool.AddTransaction(nodes[0].Blockchain, external); err != nil {
		t.Fatal(err)
	}
	block, err := nodes[0].Blockchain.MineBlock(nodes[0].Mempool, nodes[0].MinerAddress)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(block)
	if code, _ := postBlock(t, nodes[1], body); code != 200 {
		t.Fatal("expected acceptance")
	}
	if nodes[1].Mempool.Has(pending.ID) || nodes[1].Mempool.Size() != 0 {
		t.Fatal("conflicting pending spend must be revalidated away")
	}
}
func TestSendAndBroadcastBlock(t *testing.T) {
	nodes, _, _ := blockNodes(t, 3)
	block, err := nodes[0].Blockchain.MineBlock(nodes[0].Mempool, nodes[0].MinerAddress)
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewServer(nodes[1].Handler())
	defer first.Close()
	second := httptest.NewServer(nodes[2].Handler())
	defer second.Close()
	before := blockNodeSnapshot(t, nodes[0])
	if err := nodes[0].SendBlock(first.URL+"/", block); err != nil {
		t.Fatal(err)
	}
	nodes[0].Peers = []string{first.URL, second.URL}
	if err := nodes[0].BroadcastBlock(block); err != nil {
		t.Fatal(err)
	}
	nodes[0].Peers = nil
	if !bytes.Equal(before, blockNodeSnapshot(t, nodes[0])) {
		t.Fatal("sending must not mutate sender state")
	}
	for _, n := range nodes[1:] {
		if !bytes.Equal(n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1].Hash, block.Hash) {
			t.Fatal("all peers must receive block")
		}
	}
}
func TestBroadcastBlockContinuesAfterUnreachablePeer(t *testing.T) {
	nodes, _, _ := blockNodes(t, 2)
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	working := httptest.NewServer(nodes[1].Handler())
	defer working.Close()
	nodes[0].Peers = []string{closed.URL, working.URL}
	block, err := nodes[0].Blockchain.MineBlock(nodes[0].Mempool, nodes[0].MinerAddress)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodes[0].BroadcastBlock(block); err == nil {
		t.Fatal("expected broadcast error")
	}
	if !bytes.Equal(nodes[1].Blockchain.Blocks[len(nodes[1].Blockchain.Blocks)-1].Hash, block.Hash) {
		t.Fatal("working peer must still receive block")
	}
}
func TestMineAndBroadcastBlock(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			nodes, tx, _ := blockNodes(t, 2)
			for _, n := range nodes {
				if err := n.Mempool.AddTransaction(n.Blockchain, tx); err != nil {
					t.Fatal(err)
				}
			}
			server := httptest.NewServer(nodes[1].Handler())
			defer server.Close()
			if fail {
				server.Close()
			}
			nodes[0].Peers = []string{server.URL}
			length := len(nodes[0].Blockchain.Blocks)
			block, err := nodes[0].MineAndBroadcastBlock()
			if block == nil || (err != nil) != fail {
				t.Fatalf("unexpected mining result: %v %v", block, err)
			}
			if len(nodes[0].Blockchain.Blocks) != length+1 || block != nodes[0].Blockchain.Blocks[length] || nodes[0].Mempool.Size() != 0 {
				t.Fatal("local mining result and cleanup must remain committed")
			}
			if !nodes[0].Blockchain.ValidateChain() {
				t.Fatal("local chain must remain valid")
			}
			if !fail && (!bytes.Equal(nodes[1].Blockchain.Blocks[length].Hash, block.Hash) || nodes[1].Mempool.Size() != 0) {
				t.Fatal("peer must accept block and clean mempool")
			}
		})
	}
}
func TestMineAndBroadcastRejectsInvalidStateWithoutSending(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	for _, n := range []*Node{nil, NewNode(nil, NewMempool(), "Miner", nil), NewNode(NewBlockchain(), nil, "Miner", nil), NewNode(NewBlockchain(), NewMempool(), "", nil), NewNode(&Blockchain{}, NewMempool(), "Miner", nil)} {
		if n != nil {
			n.Peers = []string{server.URL}
		}
		if block, err := n.MineAndBroadcastBlock(); err == nil || block != nil {
			t.Fatal("invalid local mining must return error and no block")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("local mining failure must not broadcast")
	}
}
func TestBlockSendingErrors(t *testing.T) {
	n := &Node{}
	var nilNode *Node
	block := &Block{}
	if nilNode.SendBlock("http://example.invalid", block) == nil || n.SendBlock("", block) == nil || n.SendBlock("http://example.invalid", nil) == nil {
		t.Fatal("invalid arguments must fail")
	}
	if nilNode.BroadcastBlock(block) == nil || n.BroadcastBlock(nil) == nil {
		t.Fatal("invalid broadcast must fail")
	}
	if n.BroadcastBlock(block) != nil {
		t.Fatal("no peers must succeed")
	}
	for _, status := range []int{204, 400, 500, 307} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/block" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("incorrect outbound request")
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			if err := n.SendBlock(server.URL, block); (err == nil) != (status >= 200 && status < 300) {
				t.Fatalf("unexpected HTTP %d result: %v", status, err)
			}
		})
	}
}
