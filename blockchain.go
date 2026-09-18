package main

import (
	"bytes"
	"testing"
)

type Blockchain struct {
	Blocks []*Block
}

func NewBlockchain() *Blockchain {
	genesis := NewBlock(
		0,
		[]byte("Genesis Block"),
		[]byte{},
	)

	return &Blockchain{
		Blocks: []*Block{genesis},
	}
}
func (bc *Blockchain) AddBlock(data []byte) {
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	newBlock := NewBlock(
		lastBlock.Height+1,
		data,
		lastBlock.Hash,
	)

	bc.Blocks = append(bc.Blocks, newBlock)
}
func TestAddBlockLinksToPreviousBlock(t *testing.T) {
	blockchain := NewBlockchain()

	blockchain.AddBlock([]byte("Alice pays Bob 10"))

	if len(blockchain.Blocks) != 2 {
		t.Fatalf(
			"expected blockchain to contain 2 blocks, got %d",
			len(blockchain.Blocks),
		)
	}

	genesis := blockchain.Blocks[0]
	block1 := blockchain.Blocks[1]

	if block1.Height != 1 {
		t.Fatalf(
			"expected new block height to be 1, got %d",
			block1.Height,
		)
	}

	if !bytes.Equal(block1.PrevHash, genesis.Hash) {
		t.Fatal("new block previous hash should equal genesis hash")
	}
}
func (bc *Blockchain) ValidateChain() bool {
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		previousBlock := bc.Blocks[i-1]

		if !bytes.Equal(currentBlock.PrevHash, previousBlock.Hash) {
			return false
		}

		recalculatedHash := currentBlock.calculateHash()

		if !bytes.Equal(currentBlock.Hash, recalculatedHash) {
			return false
		}
	}

	return true
}
