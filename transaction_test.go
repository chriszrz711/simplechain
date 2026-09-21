package main

import (
	"bytes"
	"testing"
)

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	if tx.From != "Alice" {
		t.Fatalf("expected From Alice, got %s", tx.From)
	}

	if tx.To != "Bob" {
		t.Fatalf("expected To Bob, got %s", tx.To)
	}

	if tx.Amount != 10 {
		t.Fatalf("expected Amount 10, got %d", tx.Amount)
	}
}
func TestTransactionHasID(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	if len(tx.ID) == 0 {
		t.Fatal("expected transaction ID to be set")
	}
}
func TestSameTransactionHasSameID(t *testing.T) {
	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Alice", "Bob", 10)

	if !bytes.Equal(tx1.ID, tx2.ID) {
		t.Fatal("expected identical transactions to have the same ID")
	}
}
func TestDifferentTransactionHasDifferentID(t *testing.T) {
	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Alice", "Bob", 11)

	if bytes.Equal(tx1.ID, tx2.ID) {
		t.Fatal("expected different transactions to have different IDs")
	}
}
func TestNewTXOutput(t *testing.T) {
	out := NewTXOutput(10, "Bob")

	if out.Value != 10 {
		t.Fatalf("expected value 10, got %d", out.Value)
	}

	if out.To != "Bob" {
		t.Fatalf("expected owner Bob, got %s", out.To)
	}
}
func TestNewTXInput(t *testing.T) {
	txID := []byte("previous-transaction")

	in := NewTXInput(txID, 0, "Alice")

	if !bytes.Equal(in.TxID, txID) {
		t.Fatal("expected input TxID to match")
	}

	if in.OutIndex != 0 {
		t.Fatalf("expected output index 0, got %d", in.OutIndex)
	}

	if in.From != "Alice" {
		t.Fatalf("expected from Alice, got %s", in.From)
	}
}
func TestTransactionHasInputsAndOutputs(t *testing.T) {
	in := NewTXInput([]byte("AAA"), 0, "Alice")

	out1 := NewTXOutput(10, "Bob")
	out2 := NewTXOutput(40, "Alice")

	tx := Transaction{
		Inputs: []TXInput{
			*in,
		},
		Outputs: []TXOutput{
			*out1,
			*out2,
		},
	}

	if len(tx.Inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(tx.Inputs))
	}

	if len(tx.Outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(tx.Outputs))
	}

	if tx.Outputs[0].Value != 10 {
		t.Fatalf("expected first output value 10, got %d", tx.Outputs[0].Value)
	}

	if tx.Outputs[1].Value != 40 {
		t.Fatalf("expected change output value 40, got %d", tx.Outputs[1].Value)
	}
}
func TestTransactionIDChangesWithInput(t *testing.T) {
	tx1 := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     []byte("AAA"),
				OutIndex: 0,
				From:     "Alice",
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    "Bob",
			},
		},
	}

	tx2 := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     []byte("FFF"),
				OutIndex: 0,
				From:     "Alice",
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    "Bob",
			},
		},
	}

	tx1.SetID()
	tx2.SetID()

	if bytes.Equal(tx1.ID, tx2.ID) {
		t.Fatal("expected different inputs to produce different transaction IDs")
	}
}
func TestTransactionIDChangesWithOutput(t *testing.T) {
	tx1 := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     []byte("AAA"),
				OutIndex: 0,
				From:     "Alice",
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    "Bob",
			},
			{
				Value: 40,
				To:    "Alice",
			},
		},
	}

	tx2 := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     []byte("AAA"),
				OutIndex: 0,
				From:     "Alice",
			},
		},
		Outputs: []TXOutput{
			{
				Value: 20,
				To:    "Bob",
			},
			{
				Value: 30,
				To:    "Alice",
			},
		},
	}

	tx1.SetID()
	tx2.SetID()

	if bytes.Equal(tx1.ID, tx2.ID) {
		t.Fatal("expected different outputs to produce different transaction IDs")
	}
}
func TestFindUTXO(t *testing.T) {
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    "Alice",
			},
		},
	}
	funding.SetID()

	spend := Transaction{
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     "Alice",
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    "Bob",
			},
			{
				Value: 40,
				To:    "Alice",
			},
		},
	}
	spend.SetID()

	utxos := FindUTXO(
		[]Transaction{funding, spend},
		"Alice",
	)

	if len(utxos) != 1 {
		t.Fatalf("expected 1 UTXO, got %d", len(utxos))
	}

	if utxos[0].Output.Value != 40 {
		t.Fatalf(
			"expected UTXO value 40, got %d",
			utxos[0].Output.Value,
		)
	}

	if !bytes.Equal(utxos[0].TxID, spend.ID) {
		t.Fatal("expected UTXO to come from spend transaction")
	}

	if utxos[0].OutIndex != 1 {
		t.Fatalf(
			"expected output index 1, got %d",
			utxos[0].OutIndex,
		)
	}
}
func TestGetBalance(t *testing.T) {
	tx1 := Transaction{
		Outputs: []TXOutput{
			{
				Value: 20,
				To:    "Alice",
			},
			{
				Value: 10,
				To:    "Bob",
			},
		},
	}
	tx1.SetID()

	tx2 := Transaction{
		Outputs: []TXOutput{
			{
				Value: 30,
				To:    "Alice",
			},
		},
	}
	tx2.SetID()

	transactions := []Transaction{
		tx1,
		tx2,
	}

	balance := GetBalance(transactions, "Alice")

	if balance != 50 {
		t.Fatalf("expected Alice balance 50, got %d", balance)
	}
}
func TestGetBalanceAfterSpending(t *testing.T) {
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    "Alice",
			},
		},
	}
	funding.SetID()

	spend := Transaction{
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     "Alice",
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    "Bob",
			},
			{
				Value: 40,
				To:    "Alice",
			},
		},
	}
	spend.SetID()

	transactions := []Transaction{
		funding,
		spend,
	}

	balance := GetBalance(transactions, "Alice")

	if balance != 40 {
		t.Fatalf("expected Alice balance 40, got %d", balance)
	}
}
func TestFindSpendableUTXO(t *testing.T) {
	tx1 := Transaction{
		Outputs: []TXOutput{
			{
				Value: 20,
				To:    "Alice",
			},
		},
	}
	tx1.SetID()

	tx2 := Transaction{
		Outputs: []TXOutput{
			{
				Value: 30,
				To:    "Alice",
			},
		},
	}
	tx2.SetID()

	tx3 := Transaction{
		Outputs: []TXOutput{
			{
				Value: 40,
				To:    "Alice",
			},
		},
	}
	tx3.SetID()

	transactions := []Transaction{
		tx1,
		tx2,
		tx3,
	}

	total, selected := FindSpendableUTXO(
		transactions,
		"Alice",
		35,
	)

	if total != 50 {
		t.Fatalf("expected selected total 50, got %d", total)
	}

	if len(selected) != 2 {
		t.Fatalf("expected 2 selected UTXOs, got %d", len(selected))
	}
}
func TestNewUTXOTransaction(t *testing.T) {
	tx1 := Transaction{
		Outputs: []TXOutput{
			{
				Value: 20,
				To:    "Alice",
			},
		},
	}
	tx1.SetID()

	tx2 := Transaction{
		Outputs: []TXOutput{
			{
				Value: 30,
				To:    "Alice",
			},
		},
	}
	tx2.SetID()

	transactions := []Transaction{
		tx1,
		tx2,
	}

	tx, err := NewUTXOTransaction(
		"Alice",
		"Bob",
		35,
		transactions,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tx.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(tx.Inputs))
	}

	if len(tx.Outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(tx.Outputs))
	}

	if tx.Outputs[0].Value != 35 {
		t.Fatalf(
			"expected payment output 35, got %d",
			tx.Outputs[0].Value,
		)
	}

	if tx.Outputs[0].To != "Bob" {
		t.Fatalf(
			"expected payment to Bob, got %s",
			tx.Outputs[0].To,
		)
	}

	if tx.Outputs[1].Value != 15 {
		t.Fatalf(
			"expected change output 15, got %d",
			tx.Outputs[1].Value,
		)
	}

	if tx.Outputs[1].To != "Alice" {
		t.Fatalf(
			"expected change to Alice, got %s",
			tx.Outputs[1].To,
		)
	}

	if len(tx.ID) == 0 {
		t.Fatal("expected transaction ID to be set")
	}
}
func TestNewUTXOTransactionInsufficientFunds(t *testing.T) {
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 30,
				To:    "Alice",
			},
		},
	}
	funding.SetID()

	transactions := []Transaction{
		funding,
	}

	tx, err := NewUTXOTransaction(
		"Alice",
		"Bob",
		50,
		transactions,
	)

	if err == nil {
		t.Fatal("expected insufficient funds error")
	}

	if tx != nil {
		t.Fatal("expected transaction to be nil")
	}
}
func TestBalancesAfterTransaction(t *testing.T) {
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    "Alice",
			},
		},
	}
	funding.SetID()

	transactions := []Transaction{
		funding,
	}

	tx, err := NewUTXOTransaction(
		"Alice",
		"Bob",
		35,
		transactions,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	transactions = append(transactions, *tx)

	aliceBalance := GetBalance(transactions, "Alice")
	bobBalance := GetBalance(transactions, "Bob")

	if aliceBalance != 15 {
		t.Fatalf(
			"expected Alice balance 15, got %d",
			aliceBalance,
		)
	}

	if bobBalance != 35 {
		t.Fatalf(
			"expected Bob balance 35, got %d",
			bobBalance,
		)
	}
}
func TestChainedTransactions(t *testing.T) {
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    "Alice",
			},
		},
	}
	funding.SetID()

	transactions := []Transaction{
		funding,
	}

	// Transaction 1:
	// Alice -> Bob 35
	tx1, err := NewUTXOTransaction(
		"Alice",
		"Bob",
		35,
		transactions,
	)

	if err != nil {
		t.Fatalf("unexpected error creating tx1: %v", err)
	}

	transactions = append(transactions, *tx1)

	// Transaction 2:
	// Bob -> Carol 20
	tx2, err := NewUTXOTransaction(
		"Bob",
		"Carol",
		20,
		transactions,
	)

	if err != nil {
		t.Fatalf("unexpected error creating tx2: %v", err)
	}

	transactions = append(transactions, *tx2)

	aliceBalance := GetBalance(transactions, "Alice")
	bobBalance := GetBalance(transactions, "Bob")
	carolBalance := GetBalance(transactions, "Carol")

	if aliceBalance != 15 {
		t.Fatalf(
			"expected Alice balance 15, got %d",
			aliceBalance,
		)
	}

	if bobBalance != 15 {
		t.Fatalf(
			"expected Bob balance 15, got %d",
			bobBalance,
		)
	}

	if carolBalance != 20 {
		t.Fatalf(
			"expected Carol balance 20, got %d",
			carolBalance,
		)
	}
}
