package main

import (
	"bytes"
	"testing"
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
	for i, currentBlock := range bc.Blocks {
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
func TestTwoBlockchainsHaveSameGenesisHash(t *testing.T) {
	blockchain1 := NewBlockchain()
	blockchain2 := NewBlockchain()

	genesis1 := blockchain1.Blocks[0]
	genesis2 := blockchain2.Blocks[0]

	if !bytes.Equal(genesis1.Hash, genesis2.Hash) {
		t.Fatal("two blockchains should have identical genesis hashes")
	}
}
