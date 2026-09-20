package main

import (
	"bytes"
	"math/big"
)

const targetBits = 16

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

func NewProofOfWork(block *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	return &ProofOfWork{
		block:  block,
		target: target,
	}
}
func (pow *ProofOfWork) Run() (uint64, []byte) {
	var nonce uint64 = 0

	for {
		pow.block.Nonce = nonce

		hash := pow.block.calculateHash()

		var hashInt big.Int
		hashInt.SetBytes(hash)

		if hashInt.Cmp(pow.target) == -1 {
			return nonce, hash
		}

		nonce++
	}
}
func (pow *ProofOfWork) Validate() bool {
	recalculatedHash := pow.block.calculateHash()

	if !bytes.Equal(recalculatedHash, pow.block.Hash) {
		return false
	}

	var hashInt big.Int
	hashInt.SetBytes(recalculatedHash)

	return hashInt.Cmp(pow.target) == -1
}
