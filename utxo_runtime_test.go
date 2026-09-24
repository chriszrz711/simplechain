package main

import (
	"fmt"
	"maps"
	"testing"
)

func TestNewUTXOTransactionFromSetRejectsInvalidRequests(t *testing.T) {
	for _, tc := range []struct {
		name, to string
		amount   int
	}{
		{"insufficient", "Bob", 51}, {"zero", "Bob", 0}, {"negative", "Bob", -1}, {"empty_recipient", "", 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := map[string]TXOutput{"01:0": {Value: 50, To: "Alice"}}
			before := maps.Clone(set)
			tx, err := NewUTXOTransactionFromSet("Alice", tc.to, tc.amount, set)
			if err == nil || tx != nil {
				t.Fatal("expected an error and no transaction")
			}
			if !maps.Equal(set, before) {
				t.Fatal("creation must not mutate UTXO state")
			}
		})
	}
}

func TestNewSignedUTXOTransactionFromSet(t *testing.T) {
	alice, bob := NewWallet(), NewWallet()
	set := map[string]TXOutput{"01:0": {Value: 20, To: alice.Address()}, "02:1": {Value: 30, To: alice.Address()}}
	before := maps.Clone(set)
	tx, err := NewSignedUTXOTransactionFromSet(alice, bob.Address(), 50, set)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.Inputs) != 2 || len(tx.Outputs) != 1 {
		t.Fatal("expected two inputs and exact payment without change")
	}
	if !tx.VerifySignatures() || !tx.ValidateWithUTXOSet(set) {
		t.Fatal("expected valid signed transaction")
	}
	if !maps.Equal(set, before) {
		t.Fatal("creation must not consume UTXOs")
	}
	for _, tc := range []struct {
		name   string
		wallet *Wallet
		to     string
		amount int
	}{
		{"nil_wallet", nil, bob.Address(), 10}, {"insufficient", alice, bob.Address(), 51},
		{"zero", alice, bob.Address(), 0}, {"negative", alice, bob.Address(), -1}, {"empty_recipient", alice, "", 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := NewSignedUTXOTransactionFromSet(tc.wallet, tc.to, tc.amount, set)
			if err == nil || tx != nil {
				t.Fatal("expected an error and no transaction")
			}
		})
	}
}

func TestBlockchainRuntimeUsesMaintainedUTXOSet(t *testing.T) {
	bc := NewBlockchain()
	alice, bob := NewWallet(), NewWallet()
	funding := NewCoinbaseTransaction(alice.Address(), CoinbaseReward)
	if err := bc.AddBlock([]Transaction{*funding}); err != nil {
		t.Fatal(err)
	}
	before := maps.Clone(bc.UTXOSet)
	tx, err := bc.NewSignedTransaction(alice, bob.Address(), 30)
	if err != nil || tx == nil {
		t.Fatalf("failed to create payment: %v", err)
	}
	if !tx.ValidateWithUTXOSet(bc.UTXOSet) {
		t.Fatal("payment must be valid")
	}
	if !maps.Equal(before, bc.UTXOSet) {
		t.Fatal("creating payment must not consume UTXOs")
	}
	delete(bc.UTXOSet, fmt.Sprintf("%x:%d", funding.ID, 0))
	if tx, err := bc.NewSignedTransaction(alice, bob.Address(), 30); err == nil || tx != nil {
		t.Fatal("must use cached funds rather than historical funding")
	}
	if bc.GetBalance(alice.Address()) != 0 {
		t.Fatal("balance must use cached state")
	}
	if !bc.ValidateChain() {
		t.Fatal("historical audit must remain independent of cached UTXOs")
	}
	bc.UTXOSet = bc.BuildUTXOSet()
	if bc.GetBalance(alice.Address()) != CoinbaseReward {
		t.Fatal("rebuild should restore balance")
	}
	if bc.GetBalance(bob.Address()) != 0 {
		t.Fatal("unfunded owner must have zero balance")
	}
}

func TestBlockchainGetBalanceSumsOwnedOutputs(t *testing.T) {
	bc := &Blockchain{UTXOSet: map[string]TXOutput{
		"01:0": {Value: 20, To: "Alice"},
		"02:1": {Value: 15, To: "Alice"},
		"03:0": {Value: 10, To: "Bob"},
	}}
	for owner, want := range map[string]int{"Alice": 35, "Bob": 10, "Charlie": 0} {
		if got := bc.GetBalance(owner); got != want {
			t.Errorf("balance for %s: got %d, want %d", owner, got, want)
		}
	}
	if got := (&Blockchain{}).GetBalance("Alice"); got != 0 {
		t.Fatalf("nil UTXO set balance: got %d, want 0", got)
	}
}
