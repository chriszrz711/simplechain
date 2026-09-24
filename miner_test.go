package main

import (
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"testing"
)

func TestMineBlockIncludesPendingTransaction(t *testing.T) {
	bc := NewBlockchain()
	alice, bob, miner := NewWallet(), NewWallet(), NewWallet()
	if err := bc.AddBlock([]Transaction{*NewCoinbaseTransaction(alice.Address(), CoinbaseReward)}); err != nil {
		t.Fatal(err)
	}
	tx, err := bc.NewSignedTransaction(alice, bob.Address(), 30)
	if err != nil {
		t.Fatal(err)
	}
	mp := NewMempool()
	if err := mp.AddTransaction(bc, tx); err != nil {
		t.Fatal(err)
	}
	if mp.Size() != 1 {
		t.Fatal("expected one pending transaction")
	}
	length := len(bc.Blocks)
	tip := bc.Blocks[length-1]
	rewardBefore := bc.GetBalance(miner.Address())
	block, err := bc.MineBlock(mp, miner.Address())
	if err != nil {
		t.Fatal(err)
	}
	if len(bc.Blocks) != length+1 || block != bc.Blocks[length] {
		t.Fatal("returned block must be the newly appended tip")
	}
	if block.Height != tip.Height+1 || !bytes.Equal(block.PrevHash, tip.Hash) {
		t.Fatal("new block must extend the old tip")
	}
	if len(block.Transactions) != 2 || !block.Transactions[0].IsCoinbase() || !block.Transactions[0].ValidateCoinbase() {
		t.Fatal("expected valid coinbase first, followed by payment")
	}
	if !bytes.Equal(block.Transactions[1].ID, tx.ID) {
		t.Fatal("pending payment must follow coinbase")
	}
	if bc.GetBalance(miner.Address()) != rewardBefore+CoinbaseReward || bc.GetBalance(alice.Address()) != 20 || bc.GetBalance(bob.Address()) != 30 {
		t.Fatal("unexpected miner/payment/change balances")
	}
	if mp.Size() != 0 || mp.Has(tx.ID) {
		t.Fatal("confirmed payment must be removed")
	}
	if !NewProofOfWork(block).Validate() || !bc.ValidateChain() {
		t.Fatal("mined block and chain must be valid")
	}
	if !maps.Equal(bc.UTXOSet, bc.BuildUTXOSet()) {
		t.Fatal("cached UTXOs must match history")
	}
}

func TestMineBlockEmptyMempool(t *testing.T) {
	bc := NewBlockchain()
	miner := NewWallet()
	block, err := bc.MineBlock(NewMempool(), miner.Address())
	if err != nil {
		t.Fatal(err)
	}
	if block != bc.Blocks[len(bc.Blocks)-1] || len(block.Transactions) != 1 || !block.Transactions[0].IsCoinbase() {
		t.Fatal("expected coinbase-only tip")
	}
	if bc.GetBalance(miner.Address()) != CoinbaseReward || !bc.ValidateChain() {
		t.Fatal("expected reward and valid chain")
	}
}

func TestMineBlockRevalidatesStalePendingTransactions(t *testing.T) {
	bc, alice, bob, pending, _ := mempoolFixture(t)
	mp := NewMempool()
	if err := mp.AddTransaction(bc, pending); err != nil {
		t.Fatal(err)
	}
	external, err := bc.NewSignedTransaction(alice, bob.Address(), 12)
	if err != nil {
		t.Fatal(err)
	}
	if err := bc.AddBlock([]Transaction{*external}); err != nil {
		t.Fatal(err)
	}
	block, err := bc.MineBlock(mp, NewWallet().Address())
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Transactions) != 1 || !block.Transactions[0].IsCoinbase() || mp.Has(pending.ID) || mp.Size() != 0 {
		t.Fatal("stale pending transaction must be removed before mining")
	}
	if !bc.ValidateChain() {
		t.Fatal("chain must remain valid")
	}
}

func TestMineBlockUsesDeterministicSelection(t *testing.T) {
	bc, _, _, first, second := mempoolFixture(t)
	mp := NewMempool()
	for _, tx := range []*Transaction{second, first} {
		if err := mp.AddTransaction(bc, tx); err != nil {
			t.Fatal(err)
		}
	}
	selected := mp.TransactionsForBlock()
	block, err := bc.MineBlock(mp, NewWallet().Address())
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Transactions) != 3 {
		t.Fatal("expected coinbase and both pending transactions")
	}
	for i := range selected {
		if !bytes.Equal(selected[i].ID, block.Transactions[i+1].ID) {
			t.Fatal("mining must preserve deterministic mempool order")
		}
	}
	if mp.Size() != 0 || !bc.ValidateChain() {
		t.Fatal("expected empty mempool and valid chain")
	}
}

func TestMineBlockRejectsInvalidArgumentsWithoutMutation(t *testing.T) {
	for _, name := range []string{"nil_blockchain", "nil_mempool", "empty_address", "empty_chain", "nil_tip", "nil_utxo_set"} {
		t.Run(name, func(t *testing.T) {
			bc, _, _, tx, _ := mempoolFixture(t)
			mp := NewMempool()
			if err := mp.AddTransaction(bc, tx); err != nil {
				t.Fatal(err)
			}
			receiver, pool, address := bc, mp, "Miner"
			switch name {
			case "nil_blockchain":
				receiver = nil
			case "nil_mempool":
				pool = nil
			case "empty_address":
				address = ""
			case "empty_chain":
				bc.Blocks = nil
			case "nil_tip":
				bc.Blocks[len(bc.Blocks)-1] = nil
			case "nil_utxo_set":
				bc.UTXOSet = nil
			}
			blocks := append([]*Block(nil), bc.Blocks...)
			utxos := maps.Clone(bc.UTXOSet)
			pending := mp.TransactionsForBlock()
			block, err := receiver.MineBlock(pool, address)
			if err == nil || block != nil {
				t.Fatal("expected error and no block")
			}
			if !reflect.DeepEqual(blocks, bc.Blocks) || !maps.Equal(utxos, bc.UTXOSet) || !reflect.DeepEqual(pending, mp.TransactionsForBlock()) {
				t.Fatal("invalid arguments must not mutate chain or mempool")
			}
		})
	}
}

func TestMineBlockPreservesPendingWhenAddBlockFails(t *testing.T) {
	bc := NewBlockchain()
	miner := NewWallet()
	coinbase := NewCoinbaseTransaction(miner.Address(), CoinbaseReward)
	// Controlled cache fixture: the new Coinbase will replace this output with
	// CoinbaseReward, so the otherwise valid payment will fail amount validation.
	bc.UTXOSet[fmt.Sprintf("%x:0", coinbase.ID)] = TXOutput{Value: CoinbaseReward - 1, To: miner.Address()}
	tx, err := bc.NewSignedTransaction(miner, "Bob", CoinbaseReward-1)
	if err != nil {
		t.Fatal(err)
	}
	mp := NewMempool()
	if err := mp.AddTransaction(bc, tx); err != nil {
		t.Fatal(err)
	}
	before := maps.Clone(bc.UTXOSet)
	pending := mp.TransactionsForBlock()
	tip := bc.Blocks[0]
	block, err := bc.MineBlock(mp, miner.Address())
	if err == nil || block != nil {
		t.Fatal("expected AddBlock validation failure")
	}
	if len(bc.Blocks) != 1 || bc.Blocks[0] != tip || !maps.Equal(before, bc.UTXOSet) {
		t.Fatal("failed mining must preserve chain state")
	}
	if !reflect.DeepEqual(pending, mp.TransactionsForBlock()) {
		t.Fatal("failed mining must preserve valid pending transactions")
	}
}

func TestMineBlockRewardAddsToExistingBalance(t *testing.T) {
	bc := NewBlockchain()
	alice, miner := NewWallet(), NewWallet()
	if err := bc.AddBlock([]Transaction{*NewCoinbaseTransaction(alice.Address(), CoinbaseReward)}); err != nil {
		t.Fatal(err)
	}
	payment, err := bc.NewSignedTransaction(alice, miner.Address(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if err := bc.AddBlock([]Transaction{*payment}); err != nil {
		t.Fatal(err)
	}
	if _, err := bc.MineBlock(NewMempool(), miner.Address()); err != nil {
		t.Fatal(err)
	}
	if got := bc.GetBalance(miner.Address()); got != 7+CoinbaseReward {
		t.Fatalf("expected prior balance plus reward %d, got %d", 7+CoinbaseReward, got)
	}
	if !bc.ValidateChain() || !maps.Equal(bc.UTXOSet, bc.BuildUTXOSet()) {
		t.Fatal("chain and cached UTXOs must remain consistent")
	}
}
