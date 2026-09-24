package main

import (
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"testing"
)

func TestMempoolStoresTransactionByID(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()
	bob := NewWallet()

	// Alice 先获得 50
	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf(
			"failed to add coinbase block: %v",
			err,
		)
	}

	// 创建一笔正常签名交易，但暂时不 AddBlock
	tx, err := bc.NewSignedTransaction(
		alice,
		bob.Address(),
		30,
	)
	if err != nil {
		t.Fatalf(
			"failed to create transaction: %v",
			err,
		)
	}

	mempool := NewMempool()

	if err := mempool.AddTransaction(bc, tx); err != nil {
		t.Fatal(err)
	}

	key := fmt.Sprintf("%x", tx.ID)

	stored, exists := mempool.Transactions[key]

	if !exists {
		t.Fatal(
			"expected transaction to exist in mempool",
		)
	}

	if fmt.Sprintf("%x", stored.ID) !=
		fmt.Sprintf("%x", tx.ID) {

		t.Fatal(
			"stored transaction has wrong ID",
		)
	}
}

func mempoolFixture(t *testing.T) (*Blockchain, *Wallet, *Wallet, *Transaction, *Transaction) {
	t.Helper()
	bc := NewBlockchain()
	alice, bob := NewWallet(), NewWallet()
	for _, wallet := range []*Wallet{alice, bob} {
		if err := bc.AddBlock([]Transaction{*NewCoinbaseTransaction(wallet.Address(), CoinbaseReward)}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := bc.NewSignedTransaction(alice, bob.Address(), 10)
	if err != nil {
		t.Fatal(err)
	}
	second, err := bc.NewSignedTransaction(bob, alice.Address(), 20)
	if err != nil {
		t.Fatal(err)
	}
	return bc, alice, bob, first, second
}

func TestMempoolAdmissionAndRemoval(t *testing.T) {
	bc, alice, bob, first, second := mempoolFixture(t)
	before := maps.Clone(bc.UTXOSet)
	mp := NewMempool()
	if mp.Transactions == nil || mp.Size() != 0 || mp.Has(first.ID) {
		t.Fatal("expected initialized empty mempool")
	}
	if err := mp.AddTransaction(bc, first); err != nil {
		t.Fatal(err)
	}
	if err := mp.AddTransaction(bc, first); err != nil {
		t.Fatal(err)
	}
	if mp.Size() != 1 || !mp.Has(first.ID) {
		t.Fatal("duplicate admission must be idempotent")
	}
	conflict, err := bc.NewSignedTransaction(alice, bob.Address(), 11)
	if err != nil {
		t.Fatal(err)
	}
	if err := mp.AddTransaction(bc, conflict); err == nil {
		t.Fatal("expected pending double spend rejection")
	}
	if mp.Size() != 1 || mp.Has(conflict.ID) {
		t.Fatal("rejection must preserve pending transactions")
	}
	if err := mp.AddTransaction(bc, second); err != nil {
		t.Fatal(err)
	}
	if mp.Size() != 2 {
		t.Fatal("independent spends should coexist")
	}
	if !maps.Equal(before, bc.UTXOSet) {
		t.Fatal("admission must not mutate confirmed UTXOs")
	}
	keys := []string{fmt.Sprintf("%x", first.ID), fmt.Sprintf("%x", second.ID)}
	sort.Strings(keys)
	for i := 0; i < 10; i++ {
		got := mp.TransactionsForBlock()
		if len(got) != 2 {
			t.Fatal("selection should include every transaction")
		}
		for j := range got {
			if fmt.Sprintf("%x", got[j].ID) != keys[j] {
				t.Fatal("selection must be sorted by TxID")
			}
		}
	}
	mp.RemoveTransactions([]Transaction{*first, *NewCoinbaseTransaction("Miner", CoinbaseReward)})
	if mp.Has(first.ID) || !mp.Has(second.ID) || mp.Size() != 1 {
		t.Fatal("remove must delete only matching transactions")
	}
	mp.RemoveTransactions([]Transaction{*first, *second})
	if mp.Size() != 0 {
		t.Fatal("expected empty mempool")
	}
}

func TestMempoolRejectsInvalidAdmission(t *testing.T) {
	bc, _, _, first, _ := mempoolFixture(t)
	for _, name := range []string{"nil_transaction", "coinbase", "stale_id", "bad_signature", "missing_utxo"} {
		t.Run(name, func(t *testing.T) {
			mp := NewMempool()
			tx := *first
			tx.Inputs = append([]TXInput(nil), first.Inputs...)
			candidate := &tx
			chain := &Blockchain{UTXOSet: maps.Clone(bc.UTXOSet)}
			switch name {
			case "nil_transaction":
				candidate = nil
			case "coinbase":
				candidate = NewCoinbaseTransaction("Miner", CoinbaseReward)
			case "stale_id":
				tx.Amount++
			case "bad_signature":
				tx.Inputs[0].Signature = []byte("invalid")
			case "missing_utxo":
				chain.UTXOSet = make(map[string]TXOutput)
			}
			before := maps.Clone(chain.UTXOSet)
			if err := mp.AddTransaction(chain, candidate); err == nil {
				t.Fatal("invalid admission must return an error")
			}
			if mp.Size() != 0 || !maps.Equal(before, chain.UTXOSet) {
				t.Fatal("rejection must leave state unchanged")
			}
		})
	}
	if err := NewMempool().AddTransaction(nil, first); err == nil {
		t.Fatal("nil blockchain must be rejected")
	}
	var nilPool *Mempool
	if err := nilPool.AddTransaction(bc, first); err == nil {
		t.Fatal("nil mempool must be rejected")
	}
	if nilPool.Has(first.ID) || nilPool.Size() != 0 || len(nilPool.TransactionsForBlock()) != 0 {
		t.Fatal("nil pool queries must be safe")
	}
	nilPool.RemoveTransactions([]Transaction{*first})
	nilPool.Revalidate(bc)
	zero := &Mempool{}
	if err := zero.AddTransaction(bc, first); err != nil {
		t.Fatal(err)
	}
	zero.Revalidate(nil)
	if zero.Size() != 1 {
		t.Fatal("nil-chain revalidation must leave pending state unchanged")
	}
}

func TestMempoolRejectsUnconfirmedDependencies(t *testing.T) {
	bc, _, bob, parent, _ := mempoolFixture(t)
	mp := NewMempool()
	if err := mp.AddTransaction(bc, parent); err != nil {
		t.Fatal(err)
	}
	// Build a child against only the parent's output, which is not confirmed.
	set := map[string]TXOutput{fmt.Sprintf("%x:0", parent.ID): parent.Outputs[0]}
	child, err := NewSignedUTXOTransactionFromSet(bob, "Charlie", 5, set)
	if err != nil {
		t.Fatal(err)
	}
	if !child.ValidateWithUTXOSet(set) {
		t.Fatal("child must be valid against its unconfirmed funding")
	}
	if err := mp.AddTransaction(bc, child); err == nil {
		t.Fatal("unconfirmed outputs must not be spendable in v1")
	}
	// Even a directly injected child must be removed by revalidation.
	mp.Transactions[fmt.Sprintf("%x", child.ID)] = *child
	mp.Revalidate(bc)
	if mp.Size() != 1 || !mp.Has(parent.ID) || mp.Has(child.ID) {
		t.Fatal("revalidation must reject unconfirmed dependencies")
	}
}

func TestMempoolRevalidateAfterConfirmedSpend(t *testing.T) {
	bc, alice, bob, first, second := mempoolFixture(t)
	mp := NewMempool()
	for _, tx := range []*Transaction{first, second} {
		if err := mp.AddTransaction(bc, tx); err != nil {
			t.Fatal(err)
		}
	}
	external, err := bc.NewSignedTransaction(alice, bob.Address(), 12)
	if err != nil {
		t.Fatal(err)
	}
	if err := bc.AddBlock([]Transaction{*external}); err != nil {
		t.Fatal(err)
	}
	before := maps.Clone(bc.UTXOSet)
	mp.Revalidate(bc)
	if mp.Has(first.ID) || !mp.Has(second.ID) || mp.Size() != 1 {
		t.Fatal("revalidation must remove spent inputs and retain independent transactions")
	}
	if !maps.Equal(before, bc.UTXOSet) {
		t.Fatal("revalidation must not mutate confirmed UTXOs")
	}
}

func TestMempoolRevalidateDeterministicConflicts(t *testing.T) {
	bc, alice, bob, first, independent := mempoolFixture(t)
	second, err := bc.NewSignedTransaction(alice, bob.Address(), 11)
	if err != nil {
		t.Fatal(err)
	}
	coinbase := NewCoinbaseTransaction("Miner", CoinbaseReward)
	invalid := NewTransaction("Alice", "Bob", 10)
	mp := NewMempool()
	// Simulate conflicting/corrupt state through the existing exported map.
	for _, tx := range []*Transaction{second, first, independent, coinbase, invalid} {
		mp.Transactions[fmt.Sprintf("%x", tx.ID)] = *tx
	}
	before := maps.Clone(bc.UTXOSet)
	mp.Revalidate(bc)
	winner, loser := first, second
	if bytes.Compare(first.ID, second.ID) > 0 {
		winner, loser = second, first
	}
	if mp.Size() != 2 || !mp.Has(winner.ID) || !mp.Has(independent.ID) || mp.Has(loser.ID) {
		t.Fatal("revalidation must retain the lowest-ID conflicting transaction")
	}
	if mp.Has(coinbase.ID) || mp.Has(invalid.ID) {
		t.Fatal("revalidation must remove coinbase and invalid transactions")
	}
	if !maps.Equal(before, bc.UTXOSet) {
		t.Fatal("revalidation must not mutate confirmed UTXOs")
	}
}

func TestMempoolOwnsTransactionSnapshots(t *testing.T) {
	bc, _, _, tx, _ := mempoolFixture(t)
	mp := NewMempool()
	if err := mp.AddTransaction(bc, tx); err != nil {
		t.Fatal(err)
	}
	before := mp.TransactionsForBlock()
	tx.ID[0] ^= 1
	tx.Inputs[0].TxID[0] ^= 1
	tx.Inputs[0].Signature[0] ^= 1
	tx.Inputs[0].PublicKey[0] ^= 1
	tx.Outputs[0].Value++
	if !reflect.DeepEqual(before, mp.TransactionsForBlock()) {
		t.Fatal("caller mutation must not change admitted transaction")
	}
	selected := mp.TransactionsForBlock()
	selected[0].ID[0] ^= 1
	selected[0].Inputs[0].TxID[0] ^= 1
	selected[0].Inputs[0].Signature[0] ^= 1
	selected[0].Inputs[0].PublicKey[0] ^= 1
	selected[0].Outputs[0].Value++
	if !reflect.DeepEqual(before, mp.TransactionsForBlock()) {
		t.Fatal("selection mutation must not change pending transactions")
	}
}
