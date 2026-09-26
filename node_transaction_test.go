package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
)

func transactionNodes(t *testing.T, count int) ([]*Node, *Transaction) {
	t.Helper()
	alice, bob := NewWallet(), NewWallet()
	funding := NewCoinbaseTransaction(alice.Address(), CoinbaseReward)
	nodes := make([]*Node, count)
	for i := range nodes {
		bc := NewBlockchain()
		if err := bc.AddBlock([]Transaction{*funding}); err != nil {
			t.Fatal(err)
		}
		nodes[i] = NewNode(bc, NewMempool(), "", nil)
	}
	tx, err := nodes[0].Blockchain.NewSignedTransaction(alice, bob.Address(), 30)
	if err != nil {
		t.Fatal(err)
	}
	return nodes, tx
}

func postTransaction(t *testing.T, n *Node, body []byte) (*httptest.ResponseRecorder, TransactionResponse) {
	t.Helper()
	recorder := httptest.NewRecorder()
	n.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/transaction", bytes.NewReader(body)))
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected JSON response")
	}
	var response TransactionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return recorder, response
}

func TestTransactionHandlerAcceptsAndDeduplicates(t *testing.T) {
	nodes, tx := transactionNodes(t, 1)
	n := nodes[0]
	var calls atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer remote.Close()
	n.Peers = []string{remote.URL}
	before, _ := json.Marshal(n.Blockchain)
	body, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		recorder, response := postTransaction(t, n, body)
		if recorder.Code != 200 || !response.Accepted || response.TxID != fmt.Sprintf("%x", tx.ID) || response.Error != "" {
			t.Fatalf("unexpected response: %d %+v", recorder.Code, response)
		}
	}
	if n.Mempool.Size() != 1 || !n.Mempool.Has(tx.ID) {
		t.Fatal("expected exactly one pending transaction")
	}
	stored := n.Mempool.Transactions[fmt.Sprintf("%x", tx.ID)]
	if !reflect.DeepEqual(stored, *tx) {
		t.Fatal("JSON transport must preserve transaction and signatures")
	}
	after, _ := json.Marshal(n.Blockchain)
	if !bytes.Equal(before, after) || calls.Load() != 0 {
		t.Fatal("receipt must not modify blockchain or rebroadcast")
	}
}

func TestTransactionHandlerRejectsBadRequests(t *testing.T) {
	nodes, tx := transactionNodes(t, 1)
	n := nodes[0]
	invalid := cloneTransaction(*tx)
	invalid.Inputs[0].Signature = []byte("invalid")
	badSignature, _ := json.Marshal(invalid)
	coinbase, _ := json.Marshal(NewCoinbaseTransaction("Miner", CoinbaseReward))
	valid, _ := json.Marshal(tx)
	for _, body := range [][]byte{[]byte("{"), nil, []byte("null"), badSignature, coinbase, append(append([]byte{}, valid...), []byte(" {}")...)} {
		recorder, response := postTransaction(t, n, body)
		if recorder.Code != 400 || response.Accepted || response.Error == "" || n.Mempool.Size() != 0 {
			t.Fatalf("expected rejection: %d %+v", recorder.Code, response)
		}
	}
	recorder := httptest.NewRecorder()
	n.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/transaction", nil))
	if recorder.Code != 405 || recorder.Header().Get("Allow") != "POST" {
		t.Fatal("GET must return 405 and Allow POST")
	}
	for _, node := range []*Node{nil, NewNode(nil, NewMempool(), "", nil), NewNode(NewBlockchain(), nil, "", nil)} {
		recorder, response := postTransaction(t, node, valid)
		if recorder.Code != 500 || response.Accepted {
			t.Fatal("uninitialized nodes must return 500")
		}
	}
}

func TestSendAndBroadcastTransaction(t *testing.T) {
	nodes, tx := transactionNodes(t, 3)
	servers := []*httptest.Server{httptest.NewServer(nodes[1].Handler()), httptest.NewServer(nodes[2].Handler())}
	for _, server := range servers {
		defer server.Close()
	}
	sender := nodes[0]
	before, _ := json.Marshal(sender.Blockchain)
	if err := sender.SendTransaction(servers[0].URL+"/", tx); err != nil {
		t.Fatal(err)
	}
	if !nodes[1].Mempool.Has(tx.ID) {
		t.Fatal("peer must receive sent transaction")
	}
	sender.Peers = []string{servers[0].URL, servers[1].URL}
	if err := sender.BroadcastTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if !nodes[2].Mempool.Has(tx.ID) {
		t.Fatal("all peers must receive broadcast")
	}
	after, _ := json.Marshal(sender.Blockchain)
	if sender.Mempool.Size() != 0 || !bytes.Equal(before, after) {
		t.Fatal("sending must not mutate sender state")
	}
}

func TestBroadcastAttemptsPeersAfterFailure(t *testing.T) {
	nodes, tx := transactionNodes(t, 2)
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "rejected", 400) }))
	defer failing.Close()
	working := httptest.NewServer(nodes[1].Handler())
	defer working.Close()
	nodes[0].Peers = []string{failing.URL, working.URL}
	if err := nodes[0].BroadcastTransaction(tx); err == nil {
		t.Fatal("expected aggregate error")
	}
	if !nodes[1].Mempool.Has(tx.ID) {
		t.Fatal("failure must not prevent later peer delivery")
	}
}

func TestSubmitTransactionAdmitsBeforeBroadcast(t *testing.T) {
	nodes, tx := transactionNodes(t, 2)
	var admitted atomic.Bool
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admitted.Store(nodes[0].Mempool.Has(tx.ID))
		nodes[1].Handler().ServeHTTP(w, r)
	}))
	defer remote.Close()
	nodes[0].Peers = []string{remote.URL}
	if err := nodes[0].SubmitTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if !admitted.Load() || !nodes[0].Mempool.Has(tx.ID) || !nodes[1].Mempool.Has(tx.ID) {
		t.Fatal("local admission must precede successful peer admission")
	}
}

func TestSubmitRejectsInvalidWithoutBroadcast(t *testing.T) {
	nodes, tx := transactionNodes(t, 1)
	var calls atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer remote.Close()
	nodes[0].Peers = []string{remote.URL}
	tx.Amount++
	if err := nodes[0].SubmitTransaction(tx); err == nil {
		t.Fatal("invalid local transaction must be rejected")
	}
	if calls.Load() != 0 || nodes[0].Mempool.Size() != 0 {
		t.Fatal("invalid transaction must not enter pool or network")
	}
}

func TestSubmitKeepsLocalTransactionAfterPeerFailure(t *testing.T) {
	nodes, tx := transactionNodes(t, 1)
	remote := httptest.NewServer(http.NotFoundHandler())
	remote.Close()
	nodes[0].Peers = []string{remote.URL}
	if err := nodes[0].SubmitTransaction(tx); err == nil {
		t.Fatal("unreachable peer should return error")
	}
	if !nodes[0].Mempool.Has(tx.ID) {
		t.Fatal("peer failure must not roll back local admission")
	}
}

func TestTransactionSendingErrors(t *testing.T) {
	nodes, tx := transactionNodes(t, 1)
	n := nodes[0]
	var nilNode *Node
	if nilNode.SendTransaction("http://example.invalid", tx) == nil || n.SendTransaction("", tx) == nil || n.SendTransaction("http://example.invalid", nil) == nil {
		t.Fatal("invalid send arguments must be rejected")
	}
	if nilNode.BroadcastTransaction(tx) == nil || n.BroadcastTransaction(nil) == nil {
		t.Fatal("invalid broadcast arguments must be rejected")
	}
	if n.BroadcastTransaction(tx) != nil {
		t.Fatal("no peers must succeed")
	}
	if nilNode.SubmitTransaction(tx) == nil || (&Node{}).SubmitTransaction(tx) == nil {
		t.Fatal("uninitialized submission must fail")
	}
	for _, status := range []int{http.StatusNoContent, http.StatusBadRequest, http.StatusInternalServerError, http.StatusTemporaryRedirect} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/transaction" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("incorrect outbound request")
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			err := n.SendTransaction(server.URL, tx)
			if (err == nil) != (status >= 200 && status < 300) {
				t.Fatalf("unexpected result for status %d: %v", status, err)
			}
		})
	}
}
