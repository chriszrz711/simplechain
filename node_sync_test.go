package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func syncSource(t *testing.T, height int) *Node {
	t.Helper()
	n := NewNode(NewBlockchain(), NewMempool(), "", nil)
	for i := 1; i <= height; i++ {
		if err := n.Blockchain.AddBlock([]Transaction{*NewCoinbaseTransactionForHeight(fmt.Sprintf("miner-%d", i), CoinbaseReward, uint64(i))}); err != nil {
			t.Fatal(err)
		}
	}
	return n
}
func syncPrefix(t *testing.T, source *Node, height int) *Node {
	t.Helper()
	n := NewNode(NewBlockchain(), NewMempool(), "", nil)
	for i := 1; i <= height; i++ {
		body, _ := json.Marshal(source.Blockchain.Blocks[i])
		var block Block
		if err := json.Unmarshal(body, &block); err != nil {
			t.Fatal(err)
		}
		if err := n.Blockchain.AddBlockValidated(&block); err != nil {
			t.Fatal(err)
		}
	}
	return n
}
func TestBlocksHandler(t *testing.T) {
	n := syncSource(t, 4)
	before := blockNodeSnapshot(t, n)
	for _, tc := range []struct {
		query       string
		code, count int
		first       uint64
	}{
		{"?from_height=3", 200, 2, 3}, {"?from_height=0", 200, 5, 0}, {"?from_height=5", 200, 0, 0},
		{"", 400, 0, 0}, {"?from_height=", 400, 0, 0}, {"?from_height=-1", 400, 0, 0}, {"?from_height=abc", 400, 0, 0}, {"?from_height=18446744073709551616", 400, 0, 0},
	} {
		t.Run(tc.query, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			n.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/blocks"+tc.query, nil))
			if recorder.Code != tc.code {
				t.Fatalf("expected %d, got %d", tc.code, recorder.Code)
			}
			if tc.code == 200 {
				if recorder.Header().Get("Content-Type") != "application/json" {
					t.Fatal("expected JSON")
				}
				var response BlocksResponse
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Blocks == nil || len(response.Blocks) != tc.count {
					t.Fatal("expected block list, including [] when empty")
				}
				for i, b := range response.Blocks {
					if b.Height != tc.first+uint64(i) {
						t.Fatal("unexpected block order")
					}
				}
			}
		})
	}
	recorder := httptest.NewRecorder()
	n.Handler().ServeHTTP(recorder, httptest.NewRequest("POST", "/blocks?from_height=0", nil))
	if recorder.Code != 405 || recorder.Header().Get("Allow") != "GET" {
		t.Fatal("expected 405 with Allow GET")
	}
	for _, bad := range []*Node{nil, {}, NewNode(&Blockchain{}, nil, "", nil)} {
		recorder := httptest.NewRecorder()
		bad.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/blocks?from_height=0", nil))
		if recorder.Code != 500 {
			t.Fatal("uninitialized chain must return 500")
		}
	}
	if !bytes.Equal(before, blockNodeSnapshot(t, n)) {
		t.Fatal("serving blocks must not mutate state")
	}
}
func TestFetchStatusAndBlocks(t *testing.T) {
	nodes, tx, _ := blockNodes(t, 1)
	peer := nodes[0]
	if err := peer.Mempool.AddTransaction(peer.Blockchain, tx); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(peer.Handler())
	defer server.Close()
	local := syncSource(t, 0)
	before := blockNodeSnapshot(t, local)
	status, err := local.FetchStatus(server.URL + "/")
	if err != nil || status.Height != 1 || status.MempoolSize != 1 {
		t.Fatalf("bad status: %+v %v", status, err)
	}
	blocks, err := local.FetchBlocks(server.URL+"/", 1)
	if err != nil || len(blocks) != 1 || blocks[0].Height != 1 {
		t.Fatalf("bad blocks: %v %v", blocks, err)
	}
	blocks[0].Hash[0] ^= 1
	if bytes.Equal(blocks[0].Hash, peer.Blockchain.Blocks[1].Hash) {
		t.Fatal("fetched block must be independent")
	}
	if !bytes.Equal(before, blockNodeSnapshot(t, local)) {
		t.Fatal("fetching must not mutate local state")
	}
}
func TestSyncFromPeerCatchesUp(t *testing.T) {
	source := syncSource(t, 5)
	local := syncPrefix(t, source, 2)
	server := httptest.NewServer(source.Handler())
	defer server.Close()
	if err := local.SyncFromPeer(server.URL); err != nil {
		t.Fatal(err)
	}
	tip := local.Blockchain.Blocks[len(local.Blockchain.Blocks)-1]
	if tip.Height != 5 || !bytes.Equal(tip.Hash, source.Blockchain.Blocks[5].Hash) || !local.Blockchain.ValidateChain() {
		t.Fatal("expected valid synchronized chain")
	}
	if !maps.Equal(local.Blockchain.UTXOSet, local.Blockchain.BuildUTXOSet()) || !maps.Equal(local.Blockchain.UTXOSet, source.Blockchain.UTXOSet) {
		t.Fatal("UTXOs must match history and source")
	}
}
func TestSyncFromPeerNoOp(t *testing.T) {
	for _, remoteHeight := range []uint64{1, 2} {
		local := syncSource(t, 2)
		before := blockNodeSnapshot(t, local)
		var blocksRequested atomic.Bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/status" {
				blocksRequested.Store(true)
			}
			json.NewEncoder(w).Encode(NodeStatus{Height: remoteHeight})
		}))
		if err := local.SyncFromPeer(server.URL); err != nil {
			t.Fatal(err)
		}
		server.Close()
		if blocksRequested.Load() || !bytes.Equal(before, blockNodeSnapshot(t, local)) {
			t.Fatal("equal/behind peer must be a no-op")
		}
	}
}
func syncResponseServer(t *testing.T, height uint64, blocks []Block) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status":
			json.NewEncoder(w).Encode(NodeStatus{Height: height})
		case "/blocks":
			if r.URL.Query().Get("from_height") != "3" {
				t.Error("expected fetch from height 3")
			}
			json.NewEncoder(w).Encode(BlocksResponse{Blocks: blocks})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}
func TestSyncRejectsGapsAndIncompleteResponses(t *testing.T) {
	source := syncSource(t, 5)
	for _, tc := range []struct {
		name       string
		heights    []int
		wantHeight uint64
	}{
		{"gap", []int{3, 5}, 2}, {"out_of_order", []int{4, 3}, 2}, {"wrong_start", []int{4, 5}, 2}, {"duplicate", []int{3, 3}, 2}, {"empty", []int{}, 2}, {"incomplete", []int{3}, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local := syncPrefix(t, source, 2)
			blocks := make([]Block, 0)
			for _, h := range tc.heights {
				blocks = append(blocks, *source.Blockchain.Blocks[h])
			}
			server := syncResponseServer(t, 5, blocks)
			if err := local.SyncFromPeer(server.URL); err == nil {
				t.Fatal("expected sync error")
			}
			if local.Blockchain.Blocks[len(local.Blockchain.Blocks)-1].Height != tc.wantHeight || !local.Blockchain.ValidateChain() {
				t.Fatal("unexpected partial state")
			}
		})
	}
}
func TestSyncRejectsForkAndPreservesValidPrefix(t *testing.T) {
	source := syncSource(t, 4)
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprint(partial), func(t *testing.T) {
			local := syncPrefix(t, source, 2)
			blocks := []Block{*source.Blockchain.Blocks[3], *source.Blockchain.Blocks[4]}
			badIndex := 0
			if partial {
				badIndex = 1
			}
			blocks[badIndex].PrevHash = []byte("different fork")
			blocks[badIndex].Nonce, blocks[badIndex].Hash = NewProofOfWork(&blocks[badIndex]).Run()
			server := syncResponseServer(t, 4, blocks)
			if err := local.SyncFromPeer(server.URL); err == nil {
				t.Fatal("nonconnecting block must be rejected")
			}
			want := 3
			if partial {
				want = 4
			}
			if len(local.Blockchain.Blocks) != want || !local.Blockchain.ValidateChain() || !maps.Equal(local.Blockchain.UTXOSet, local.Blockchain.BuildUTXOSet()) {
				t.Fatal("must retain only valid prefix, without reorg")
			}
		})
	}
}
func TestSyncCleansConfirmedAndConflictingMempool(t *testing.T) {
	source := NewNode(NewBlockchain(), NewMempool(), "Miner", nil)
	alice, bob := NewWallet(), NewWallet()
	for _, wallet := range []*Wallet{alice, bob} {
		if err := source.Blockchain.AddBlock([]Transaction{*NewCoinbaseTransactionForHeight(wallet.Address(), CoinbaseReward, uint64(len(source.Blockchain.Blocks)))}); err != nil {
			t.Fatal(err)
		}
	}
	local := syncPrefix(t, source, 2)
	confirmed, err := source.Blockchain.NewSignedTransaction(alice, "Carol", 10)
	if err != nil {
		t.Fatal(err)
	}
	stale, err := source.Blockchain.NewSignedTransaction(bob, "Carol", 11)
	if err != nil {
		t.Fatal(err)
	}
	external, err := source.Blockchain.NewSignedTransaction(bob, "Dave", 12)
	if err != nil {
		t.Fatal(err)
	}
	for _, tx := range []*Transaction{confirmed, stale} {
		if err := local.Mempool.AddTransaction(local.Blockchain, tx); err != nil {
			t.Fatal(err)
		}
	}
	if err := source.Blockchain.AddBlock([]Transaction{*confirmed, *external}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(source.Handler())
	defer server.Close()
	if err := local.SyncFromPeer(server.URL); err != nil {
		t.Fatal(err)
	}
	if local.Mempool.Size() != 0 || !local.Blockchain.ValidateChain() {
		t.Fatal("sync must remove confirmed and conflicting pending transactions")
	}
}
func TestSyncPeerResponseErrors(t *testing.T) {
	n := syncSource(t, 0)
	for _, body := range []string{"{", "null", "{} {}"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		if _, err := n.FetchStatus(server.URL); err == nil {
			t.Fatal("invalid status JSON must fail")
		}
		if _, err := n.FetchBlocks(server.URL, 0); err == nil {
			t.Fatal("invalid blocks JSON must fail")
		}
		server.Close()
	}
	for _, status := range []int{400, 500, 307} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		if _, err := n.FetchStatus(server.URL); err == nil {
			t.Fatal("non-2xx status must fail")
		}
		if _, err := n.FetchBlocks(server.URL, 0); err == nil {
			t.Fatal("non-2xx blocks must fail")
		}
		server.Close()
	}
	if _, err := n.FetchStatus(""); err == nil {
		t.Fatal("empty peer must fail")
	}
	if _, err := n.FetchBlocks("://bad", 0); err == nil {
		t.Fatal("invalid peer must fail")
	}
	for _, bad := range []*Node{nil, {}, NewNode(&Blockchain{}, NewMempool(), "", nil), NewNode(NewBlockchain(), nil, "", nil)} {
		if err := bad.SyncFromPeer("http://example.invalid"); err == nil {
			t.Fatal("invalid node must fail")
		}
	}
	if err := n.SyncFromPeer(""); err == nil {
		t.Fatal("empty sync peer must fail")
	}
}
