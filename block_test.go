package main

import (
	"bytes"
	"testing"
)

func TestCalculateHashSameBlock(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	block1 := &Block{
		Height:       1,
		Timestamp:    1000,
		Transactions: []Transaction{*tx},
		PrevHash:     []byte("previous hash"),
	}

	block2 := &Block{
		Height:       1,
		Timestamp:    1000,
		Transactions: []Transaction{*tx},
		PrevHash:     []byte("previous hash"),
	}

	hash1 := block1.calculateHash()
	hash2 := block2.calculateHash()

	if !bytes.Equal(hash1, hash2) {
		t.Fatal("identical blocks should produce identical hashes")
	}
}
func TestCalculateHashDifferentTransactions(t *testing.T) {
	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Alice", "Bob", 20)

	block1 := &Block{
		Height:       1,
		Timestamp:    1000,
		Transactions: []Transaction{*tx1},
		PrevHash:     []byte("previous hash"),
	}

	block2 := &Block{
		Height:       1,
		Timestamp:    1000,
		Transactions: []Transaction{*tx2},
		PrevHash:     []byte("previous hash"),
	}

	hash1 := block1.calculateHash()
	hash2 := block2.calculateHash()

	if bytes.Equal(hash1, hash2) {
		t.Fatal("blocks with different transactions should produce different hashes")
	}
}
func TestCalculateHashDifferentPrevHash(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	block1 := &Block{
		Height:       1,
		Timestamp:    1000,
		Transactions: []Transaction{*tx},
		PrevHash:     []byte("previous hash A"),
	}

	block2 := &Block{
		Height:       1,
		Timestamp:    1000,
		Transactions: []Transaction{*tx},
		PrevHash:     []byte("previous hash B"),
	}

	hash1 := block1.calculateHash()
	hash2 := block2.calculateHash()

	if bytes.Equal(hash1, hash2) {
		t.Fatal("blocks with different previous hashes should produce different hashes")
	}
}
func TestCalculateHashLength(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	block := &Block{
		Height:       1,
		Timestamp:    1000,
		Transactions: []Transaction{*tx},
		PrevHash:     []byte("previous hash"),
	}

	hash := block.calculateHash()

	if len(hash) != 32 {
		t.Fatalf("expected hash length to be 32 bytes, got %d", len(hash))
	}
}
func TestBlockCanContainTransactions(t *testing.T) {
	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Bob", "Carol", 5)

	block := Block{
		Transactions: []Transaction{
			*tx1,
			*tx2,
		},
	}

	if len(block.Transactions) != 2 {
		t.Fatalf(
			"expected 2 transactions, got %d",
			len(block.Transactions),
		)
	}
}
func TestBlockHashChangesWhenTransactionChanges(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	block := Block{
		Height: 1,
		Transactions: []Transaction{
			*tx,
		},
		PrevHash: []byte("previous hash"),
		Nonce:    0,
	}

	hash1 := block.calculateHash()

	// 篡改交易内容
	block.Transactions[0].Amount = 1000
	block.Transactions[0].SetID()
	hash2 := block.calculateHash()

	if bytes.Equal(hash1, hash2) {
		t.Fatal("expected block hash to change when transaction changes")
	}
}
func TestNewBlockAcceptsTransactions(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	block := NewBlock(
		1,
		[]Transaction{*tx},
		[]byte("previous hash"),
	)

	if len(block.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(block.Transactions))
	}

	if block.Transactions[0].Amount != 10 {
		t.Fatalf(
			"expected transaction amount 10, got %d",
			block.Transactions[0].Amount,
		)
	}
}
