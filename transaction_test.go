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
func TestTransactionValidateID(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	if !tx.ValidateID() {
		t.Fatal("new transaction should have a valid ID")
	}

	// 篡改交易内容，但故意不重新 SetID()
	tx.Amount = 1000

	if tx.ValidateID() {
		t.Fatal("tampered transaction should have an invalid ID")
	}
}

func TestTransactionValidateIDDetectsInputTampering(t *testing.T) {
	tx := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{TxID: []byte("previous-transaction"), OutIndex: 0, From: "Alice"},
		},
		Outputs: []TXOutput{
			{Value: 10, To: "Bob"},
		},
	}
	tx.SetID()

	if !tx.ValidateID() {
		t.Fatal("transaction should have a valid ID before input tampering")
	}

	// 修改 Input，但保留原来的交易 ID。
	tx.Inputs[0].OutIndex = 1

	if tx.ValidateID() {
		t.Fatal("transaction should have an invalid ID after input tampering")
	}
}

func TestTransactionValidateIDDetectsOutputTampering(t *testing.T) {
	tx := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{TxID: []byte("previous-transaction"), OutIndex: 0, From: "Alice"},
		},
		Outputs: []TXOutput{
			{Value: 10, To: "Bob"},
		},
	}
	tx.SetID()

	if !tx.ValidateID() {
		t.Fatal("transaction should have a valid ID before output tampering")
	}

	// 修改 Output，但保留原来的交易 ID。
	tx.Outputs[0].Value = 1000

	if tx.ValidateID() {
		t.Fatal("transaction should have an invalid ID after output tampering")
	}
}
func TestTXInputCanCarrySignatureAndPublicKey(t *testing.T) {
	wallet := NewWallet()

	data := []byte("test transaction")

	signature, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("failed to sign data: %v", err)
	}

	input := TXInput{
		TxID:     []byte("previous-transaction"),
		OutIndex: 0,
		From:     "Alice",

		Signature: signature,
		PublicKey: wallet.PublicKey,
	}

	if len(input.Signature) == 0 {
		t.Fatal("expected input to contain a signature")
	}

	if len(input.PublicKey) == 0 {
		t.Fatal("expected input to contain a public key")
	}

	if !VerifySignature(
		input.PublicKey,
		data,
		input.Signature,
	) {
		t.Fatal("expected input signature to be valid")
	}
}
func TestTransactionSigningDataIsStable(t *testing.T) {
	tx := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     []byte("previous-transaction"),
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

	data1 := tx.SigningData()
	data2 := tx.SigningData()

	if !bytes.Equal(data1, data2) {
		t.Fatal("expected signing data to be deterministic")
	}

	if len(data1) == 0 {
		t.Fatal("expected signing data to be non-empty")
	}
}
func TestTransactionCanBeSigned(t *testing.T) {
	wallet := NewWallet()

	tx := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     []byte("previous-transaction"),
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

	tx.SetID()

	err := tx.Sign(wallet)
	if err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	if len(tx.Inputs[0].Signature) == 0 {
		t.Fatal("expected transaction input to contain signature")
	}

	if len(tx.Inputs[0].PublicKey) == 0 {
		t.Fatal("expected transaction input to contain public key")
	}

	if !VerifySignature(
		tx.Inputs[0].PublicKey,
		tx.SigningData(),
		tx.Inputs[0].Signature,
	) {
		t.Fatal("expected transaction signature to be valid")
	}
}
func TestTransactionVerifySignatures(t *testing.T) {
	wallet := NewWallet()

	tx := Transaction{
		From:   "Alice",
		To:     "Bob",
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     []byte("previous-transaction"),
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

	tx.SetID()

	if err := tx.Sign(wallet); err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	// 正常签名应该验证成功
	if !tx.VerifySignatures() {
		t.Fatal("expected transaction signatures to be valid")
	}

	// 签名之后篡改交易内容
	tx.Outputs[0].Value = 1000

	// 原签名应该立即失效
	if tx.VerifySignatures() {
		t.Fatal("expected transaction signature to be invalid after tampering")
	}
}
func TestTXOutputCanBeUnlockedByCorrectPublicKey(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	output := TXOutput{
		Value: 50,
		To:    aliceWallet.Address(),
	}

	if !output.CanBeUnlockedWith(aliceWallet.PublicKey) {
		t.Fatal("Alice public key should unlock Alice output")
	}

	if output.CanBeUnlockedWith(bobWallet.PublicKey) {
		t.Fatal("Bob public key should not unlock Alice output")
	}
}
func TestTransactionRejectsInputOwnedByDifferentWallet(t *testing.T) {
	aliceWallet := NewWallet()
	hackerWallet := NewWallet()

	// 以前的一笔交易：50 属于 Alice
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    aliceWallet.Address(),
			},
		},
	}
	funding.SetID()

	// Hacker 尝试花 Alice 的 funding:0
	theft := Transaction{
		From:   "Hacker",
		To:     hackerWallet.Address(),
		Amount: 50,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     "Hacker",
			},
		},
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    hackerWallet.Address(),
			},
		},
	}

	theft.SetID()

	if err := theft.Sign(hackerWallet); err != nil {
		t.Fatalf("failed to sign theft transaction: %v", err)
	}

	// Hacker 的签名本身是真的
	if !theft.VerifySignatures() {
		t.Fatal("expected hacker signature itself to be cryptographically valid")
	}

	// 但 Hacker 没有权利花 Alice 的 Output
	if theft.VerifyInputs([]Transaction{funding}) {
		t.Fatal("transaction should not be allowed to spend an output owned by Alice")
	}
}
func TestTransactionAcceptsInputOwnedByCorrectWallet(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	// 以前的一笔交易：50 属于 Alice
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    aliceWallet.Address(),
			},
		},
	}
	funding.SetID()

	// Alice 花自己的 funding:0，给 Bob 10
	spend := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    bobWallet.Address(),
			},
			{
				Value: 40,
				To:    aliceWallet.Address(),
			},
		},
	}

	spend.SetID()

	if err := spend.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	if !spend.VerifySignatures() {
		t.Fatal("expected Alice signature to be valid")
	}

	if !spend.VerifyInputs([]Transaction{funding}) {
		t.Fatal("Alice should be allowed to spend her own output")
	}
}
func TestTransactionValidate(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	// 50 属于 Alice
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    aliceWallet.Address(),
			},
		},
	}
	funding.SetID()

	// Alice 花 10 给 Bob，40 找零
	spend := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    bobWallet.Address(),
			},
			{
				Value: 40,
				To:    aliceWallet.Address(),
			},
		},
	}

	spend.SetID()

	if err := spend.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	if !spend.Validate([]Transaction{funding}) {
		t.Fatal("expected valid transaction to pass validation")
	}
}
func TestTransactionCanIdentifyCoinbase(t *testing.T) {
	aliceWallet := NewWallet()

	coinbase := Transaction{
		Coinbase: true,
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    aliceWallet.Address(),
			},
		},
	}

	if !coinbase.IsCoinbase() {
		t.Fatal("expected transaction with no inputs to be coinbase")
	}

	normal := Transaction{
		Inputs: []TXInput{
			{
				TxID:     []byte("previous transaction"),
				OutIndex: 0,
			},
		},
	}

	if normal.IsCoinbase() {
		t.Fatal("expected transaction with inputs not to be coinbase")
	}
}
func TestValidateChainRejectsTransactionSpendingAnotherWalletOutput(t *testing.T) {
	aliceWallet := NewWallet()
	hackerWallet := NewWallet()

	blockchain := NewBlockchain()

	// Block 1:
	// 创建 50 给 Alice
	funding := Transaction{
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    aliceWallet.Address(),
			},
		},
	}
	funding.SetID()

	blockchain.AddBlock([]Transaction{funding})

	// Block 2:
	// Hacker 试图花 Alice 的 funding:0
	theft := Transaction{
		From:   hackerWallet.Address(),
		To:     hackerWallet.Address(),
		Amount: 50,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     hackerWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    hackerWallet.Address(),
			},
		},
	}

	theft.SetID()

	if err := theft.Sign(hackerWallet); err != nil {
		t.Fatalf("failed to sign theft transaction: %v", err)
	}

	blockchain.AddBlock([]Transaction{theft})

	// 所有 Block 的 PoW 都是真的，
	// 但 theft 不应该通过交易所有权验证。
	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject transaction spending another wallet's output")
	}
}
func TestTransactionWithoutInputsIsNotAutomaticallyCoinbase(t *testing.T) {
	tx := Transaction{
		Outputs: []TXOutput{
			{
				Value: 1000000,
				To:    "hacker",
			},
		},
	}

	if tx.IsCoinbase() {
		t.Fatal("transaction should not be coinbase just because it has no inputs")
	}
}
func TestNewCoinbaseTransaction(t *testing.T) {
	aliceWallet := NewWallet()

	tx := NewCoinbaseTransaction(
		aliceWallet.Address(),
		50,
	)

	if !tx.IsCoinbase() {
		t.Fatal("expected transaction to be coinbase")
	}

	if len(tx.Inputs) != 0 {
		t.Fatal("expected coinbase transaction to have no inputs")
	}

	if len(tx.Outputs) != 1 {
		t.Fatalf(
			"expected coinbase transaction to have 1 output, got %d",
			len(tx.Outputs),
		)
	}

	if tx.Outputs[0].Value != 50 {
		t.Fatalf(
			"expected coinbase output value 50, got %d",
			tx.Outputs[0].Value,
		)
	}

	if tx.Outputs[0].To != aliceWallet.Address() {
		t.Fatal("expected coinbase output to belong to Alice")
	}

	if len(tx.ID) == 0 {
		t.Fatal("expected coinbase transaction to have an ID")
	}
}
func TestCoinbaseRejectsInvalidReward(t *testing.T) {
	hackerWallet := NewWallet()

	tx := NewCoinbaseTransaction(
		hackerWallet.Address(),
		1000000,
	)

	if tx.Validate(nil) {
		t.Fatal("coinbase transaction should reject reward larger than allowed")
	}
}
func TestTransactionRejectsOutputsGreaterThanInputs(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	funding := NewCoinbaseTransaction(
		aliceWallet.Address(),
		CoinbaseReward,
	)

	spend := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 1000,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 1000,
				To:    bobWallet.Address(),
			},
		},
	}

	spend.SetID()

	if err := spend.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	if spend.Validate([]Transaction{*funding}) {
		t.Fatal("transaction should reject outputs greater than inputs")
	}
}
