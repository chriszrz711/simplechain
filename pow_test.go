package main

import (
	"math/big"
	"testing"
)

func TestNewBlockProducesValidProofOfWork(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)
	block := NewBlock(
		1,
		[]Transaction{*tx},
		[]byte("previous hash"),
	)

	targetBits := 16

	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	var hashInt big.Int
	hashInt.SetBytes(block.Hash)

	if hashInt.Cmp(target) >= 0 {
		t.Fatal("new block should have a hash below the proof-of-work target")
	}
}
func TestProofOfWorkValidateReturnsTrueForMinedBlock(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)
	block := NewBlock(
		1,
		[]Transaction{*tx},
		[]byte("previous hash"),
	)

	pow := NewProofOfWork(block)

	if !pow.Validate() {
		t.Fatal("valid proof of work should pass validation")
	}
}
func TestProofOfWorkValidateDetectsTamperedHash(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)
	block := NewBlock(
		1,
		[]Transaction{*tx},
		[]byte("previous hash"),
	)

	block.Hash = []byte("fake hash")

	pow := NewProofOfWork(block)

	if pow.Validate() {
		t.Fatal("proof of work should fail when stored hash is tampered")
	}
}
func TestValidateChainRejectsInvalidProofOfWork(t *testing.T) {
	blockchain := NewBlockchain()
	tx := NewTransaction("Alice", "Bob", 10)
	blockchain.addBlockUnchecked([]Transaction{*tx})

	block := blockchain.Blocks[1]

	for {
		block.Nonce++
		block.Hash = block.calculateHash()

		pow := NewProofOfWork(block)

		if !pow.Validate() {
			break
		}
	}

	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject a block with invalid proof of work")
	}
}
func TestProofOfWorkFailsAfterTransactionTampering(t *testing.T) {
	tx := NewTransaction("Alice", "Bob", 10)

	block := &Block{
		Height: 1,
		Transactions: []Transaction{
			*tx,
		},
		PrevHash: []byte("previous hash"),
	}

	pow := NewProofOfWork(block)

	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash

	if !pow.Validate() {
		t.Fatal("expected proof of work to be valid before tampering")
	}

	// 篡改交易
	block.Transactions[0].Amount = 1000
	block.Transactions[0].SetID()

	if pow.Validate() {
		t.Fatal("expected proof of work to be invalid after transaction tampering")
	}
}
