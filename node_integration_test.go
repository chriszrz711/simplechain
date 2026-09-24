package main

import (
	"bytes"
	"maps"
	"net/http/httptest"
	"testing"
)

func TestSimplechainV1MultiNodeLifecycle(t *testing.T) {
	nodes, tx, alice := blockNodes(t, 2)
	a, b := nodes[0], nodes[1]
	serverA := httptest.NewServer(a.Handler())
	defer serverA.Close()
	serverB := httptest.NewServer(b.Handler())
	defer serverB.Close()
	a.Peers = []string{serverB.URL}
	if err := a.SubmitTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if !a.Mempool.Has(tx.ID) || !b.Mempool.Has(tx.ID) {
		t.Fatal("signed payment must propagate to both mempools")
	}
	block, err := a.MineAndBroadcastBlock()
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Transactions) != 2 || !block.Transactions[0].IsCoinbase() || !bytes.Equal(block.Transactions[1].ID, tx.ID) {
		t.Fatal("mined block must contain reward then payment")
	}
	verify := func(t *testing.T, n *Node) {
		t.Helper()
		tip := n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1]
		if tip.Height != block.Height || !bytes.Equal(tip.Hash, block.Hash) || !n.Blockchain.ValidateChain() {
			t.Fatal("node must hold the valid canonical tip")
		}
		if n.Mempool.Has(tx.ID) || n.Mempool.Size() != 0 {
			t.Fatal("confirmed transaction must leave mempool")
		}
		for owner, want := range map[string]int{alice.Address(): 20, tx.To: 30, a.MinerAddress: CoinbaseReward} {
			if got := n.Blockchain.GetBalance(owner); got != want {
				t.Fatalf("balance got %d, want %d", got, want)
			}
		}
		if !maps.Equal(n.Blockchain.UTXOSet, n.Blockchain.BuildUTXOSet()) || !maps.Equal(n.Blockchain.UTXOSet, a.Blockchain.UTXOSet) {
			t.Fatal("cached and rebuilt UTXOs must match canonical state")
		}
	}
	verify(t, a)
	verify(t, b)
	c := syncPrefix(t, a, 1)
	if c.Blockchain.Blocks[len(c.Blockchain.Blocks)-1].Height >= block.Height {
		t.Fatal("C must start behind")
	}
	if err := c.SyncFromPeer(serverA.URL); err != nil {
		t.Fatal(err)
	}
	verify(t, c)
}
