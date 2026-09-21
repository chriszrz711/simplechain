package main

import (
	"bytes"
)

type Blockchain struct {
	Blocks []*Block
}

func NewBlockchain() *Blockchain {
	genesis := NewGenesisBlock()

	return &Blockchain{
		Blocks: []*Block{genesis},
	}
}
func (bc *Blockchain) AddBlock(transactions []Transaction) {
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	newBlock := NewBlock(
		lastBlock.Height+1,
		transactions,
		lastBlock.Hash,
	)

	bc.Blocks = append(bc.Blocks, newBlock)
}

func (bc *Blockchain) ValidateChain() bool {
	if len(bc.Blocks) == 0 {
		return false
	}
	for i, currentBlock := range bc.Blocks {
		if currentBlock.Height != uint64(i) {
			return false
		}

		if i == 0 && len(currentBlock.PrevHash) != 0 {
			return false
		}

		pow := NewProofOfWork(currentBlock)

		if !pow.Validate() {
			return false
		}

		if i > 0 {
			previousBlock := bc.Blocks[i-1]

			if !bytes.Equal(currentBlock.PrevHash, previousBlock.Hash) {
				return false
			}
		}
	}

	return true
}
