package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func readNodeStatus(t *testing.T, handler http.Handler) NodeStatus {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields["height"] == nil || fields["mempool_size"] == nil {
		t.Fatalf("unexpected status fields: %v", fields)
	}
	var status NodeStatus
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestNodeStatusReturnsCurrentState(t *testing.T) {
	bc, mp, miner := NewBlockchain(), NewMempool(), NewWallet()
	node := NewNode(bc, mp, miner.Address(), nil)
	handler := node.Handler()
	status := readNodeStatus(t, handler)
	if status.Height != bc.Blocks[len(bc.Blocks)-1].Height || status.MempoolSize != 0 {
		t.Fatalf("unexpected initial status: %+v", status)
	}
	block, err := bc.MineBlock(mp, miner.Address())
	if err != nil {
		t.Fatal(err)
	}
	status = readNodeStatus(t, handler)
	if status.Height != block.Height || status.MempoolSize != 0 {
		t.Fatalf("status did not reflect mining: %+v", status)
	}
}

func TestNodeStatusReflectsMempoolWithoutMutation(t *testing.T) {
	bc, mp := NewBlockchain(), NewMempool()
	alice, bob := NewWallet(), NewWallet()
	if err := bc.AddBlock([]Transaction{*NewCoinbaseTransaction(alice.Address(), CoinbaseReward)}); err != nil {
		t.Fatal(err)
	}
	tx, err := bc.NewSignedTransaction(alice, bob.Address(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if err := mp.AddTransaction(bc, tx); err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(struct {
		Chain *Blockchain
		Pool  *Mempool
	}{bc, mp})
	if err != nil {
		t.Fatal(err)
	}
	status := readNodeStatus(t, NewNode(bc, mp, alice.Address(), nil).Handler())
	if status.MempoolSize != 1 || status.Height != bc.Blocks[len(bc.Blocks)-1].Height {
		t.Fatalf("unexpected pending status: %+v", status)
	}
	after, err := json.Marshal(struct {
		Chain *Blockchain
		Pool  *Mempool
	}{bc, mp})
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("status must not mutate blockchain or mempool")
	}
}

func TestNodeStatusRejectsNonGET(t *testing.T) {
	node := NewNode(NewBlockchain(), NewMempool(), "Miner", nil)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			node.Handler().ServeHTTP(recorder, httptest.NewRequest(method, "/status", nil))
			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected 405, got %d", recorder.Code)
			}
			if recorder.Header().Get("Allow") != http.MethodGet {
				t.Fatal("Allow must advertise GET")
			}
		})
	}
}

func TestNodeStatusRejectsUninitializedState(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *Node
	}{
		{"nil_node", nil},
		{"nil_blockchain", NewNode(nil, NewMempool(), "", nil)},
		{"empty_blockchain", NewNode(&Blockchain{}, NewMempool(), "", nil)},
		{"nil_mempool", NewNode(NewBlockchain(), nil, "", nil)},
		{"nil_tip", NewNode(&Blockchain{Blocks: []*Block{nil}}, NewMempool(), "", nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.node.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))
			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("expected 500, got %d", recorder.Code)
			}
		})
	}
}

func TestNewNodeCopiesPeers(t *testing.T) {
	bc, mp := NewBlockchain(), NewMempool()
	peers := []string{"http://node-b:8002"}
	node := NewNode(bc, mp, "Miner", peers)
	if node.Blockchain != bc || node.Mempool != mp || node.MinerAddress != "Miner" || !reflect.DeepEqual(node.Peers, peers) {
		t.Fatal("constructor must preserve supplied fields")
	}
	peers[0] = "http://changed:9000"
	if node.Peers[0] != "http://node-b:8002" {
		t.Fatal("node peers must not alias caller's slice")
	}
	node.Peers[0] = "http://node-c:8003"
	if peers[0] != "http://changed:9000" {
		t.Fatal("caller peers must not alias node's slice")
	}
}

func TestNodeHandlerRejectsUnknownRoutes(t *testing.T) {
	node := NewNode(NewBlockchain(), NewMempool(), "Miner", nil)
	for _, path := range []string{"/", "/transactions", "/unknown", "/status/extra"} {
		recorder := httptest.NewRecorder()
		node.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for %s, got %d", path, recorder.Code)
		}
	}
}
