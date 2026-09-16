package main

import (
	"bytes"
	"testing"
)

func TestCalculateHashSameBlock(t *testing.T) {
	block1 := &Block{
		Height:    1,
		Timestamp: 1000,
		Data:      []byte("Alice pays Bob 10"),
		PrevHash:  []byte("previous hash"),
	}

	block2 := &Block{
		Height:    1,
		Timestamp: 1000,
		Data:      []byte("Alice pays Bob 10"),
		PrevHash:  []byte("previous hash"),
	}

	hash1 := block1.calculateHash()
	hash2 := block2.calculateHash()

	if !bytes.Equal(hash1, hash2) {
		t.Fatal("identical blocks should produce identical hashes")
	}
}
func TestCalculateHashDifferentData(t *testing.T) {
	block1 := &Block{
		Height:    1,
		Timestamp: 1000,
		Data:      []byte("Alice pays Bob 10"),
		PrevHash:  []byte("previous hash"),
	}

	block2 := &Block{
		Height:    1,
		Timestamp: 1000,
		Data:      []byte("Alice pays Bob 11"),
		PrevHash:  []byte("previous hash"),
	}

	hash1 := block1.calculateHash()
	hash2 := block2.calculateHash()

	if bytes.Equal(hash1, hash2) {
		t.Fatal("blocks with different data should produce different hashes")
	}
}
func TestCalculateHashDifferentPrevHash(t *testing.T) {
	block1 := &Block{
		Height:    1,
		Timestamp: 1000,
		Data:      []byte("Alice pays Bob 10"),
		PrevHash:  []byte("previous hash A"),
	}

	block2 := &Block{
		Height:    1,
		Timestamp: 1000,
		Data:      []byte("Alice pays Bob 10"),
		PrevHash:  []byte("previous hash B"),
	}

	hash1 := block1.calculateHash()
	hash2 := block2.calculateHash()

	if bytes.Equal(hash1, hash2) {
		t.Fatal("blocks with different previous hashes should produce different hashes")
	}
}
func TestCalculateHashLength(t *testing.T) {
	block := &Block{
		Height:    1,
		Timestamp: 1000,
		Data:      []byte("Alice pays Bob 10"),
		PrevHash:  []byte("previous hash"),
	}

	hash := block.calculateHash()

	if len(hash) != 32 {
		t.Fatalf("expected hash length to be 32 bytes, got %d", len(hash))
	}
}
