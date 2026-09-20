package main

import (
	"math/big"
	"testing"
)

func TestNewBlockProducesValidProofOfWork(t *testing.T) {
	block := NewBlock(
		1,
		[]byte("Alice pays Bob 10"),
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
	block := NewBlock(
		1,
		[]byte("Alice pays Bob 10"),
		[]byte("previous hash"),
	)

	pow := NewProofOfWork(block)

	if !pow.Validate() {
		t.Fatal("valid proof of work should pass validation")
	}
}
func TestProofOfWorkValidateDetectsTamperedHash(t *testing.T) {
	block := NewBlock(
		1,
		[]byte("Alice pays Bob 10"),
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
	blockchain.AddBlock([]byte("Alice pays Bob 10"))

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
