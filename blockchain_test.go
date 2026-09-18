package main

import (
	"bytes"
	"testing"
)

func TestNewBlockchainCreatesGenesisBlock(t *testing.T) {
	blockchain := NewBlockchain()

	if len(blockchain.Blocks) != 1 {
		t.Fatalf(
			"expected blockchain to contain 1 block, got %d",
			len(blockchain.Blocks),
		)
	}

	genesis := blockchain.Blocks[0]

	if genesis.Height != 0 {
		t.Fatalf(
			"expected genesis height to be 0, got %d",
			genesis.Height,
		)
	}

	if len(genesis.PrevHash) != 0 {
		t.Fatal("expected genesis previous hash to be empty")
	}

	if !bytes.Equal(genesis.Data, []byte("Genesis Block")) {
		t.Fatalf(
			"expected genesis data to be %q, got %q",
			"Genesis Block",
			genesis.Data,
		)
	}
}
func TestAddMultipleBlocksLinksCorrectly(t *testing.T) {
	blockchain := NewBlockchain()

	blockchain.AddBlock([]byte("Alice pays Bob 10"))
	blockchain.AddBlock([]byte("Bob pays Charlie 5"))

	if len(blockchain.Blocks) != 3 {
		t.Fatalf(
			"expected blockchain to contain 3 blocks, got %d",
			len(blockchain.Blocks),
		)
	}

	genesis := blockchain.Blocks[0]
	block1 := blockchain.Blocks[1]
	block2 := blockchain.Blocks[2]

	if !bytes.Equal(block1.PrevHash, genesis.Hash) {
		t.Fatal("block 1 previous hash should equal genesis hash")
	}

	if !bytes.Equal(block2.PrevHash, block1.Hash) {
		t.Fatal("block 2 previous hash should equal block 1 hash")
	}

	if block2.Height != 2 {
		t.Fatalf(
			"expected block 2 height to be 2, got %d",
			block2.Height,
		)
	}
}
func TestValidateChainReturnsTrueForValidChain(t *testing.T) {
	blockchain := NewBlockchain()

	blockchain.AddBlock([]byte("Alice pays Bob 10"))
	blockchain.AddBlock([]byte("Bob pays Charlie 5"))

	if !blockchain.ValidateChain() {
		t.Fatal("expected valid blockchain to pass validation")
	}
}
func TestValidateChainDetectsTamperedData(t *testing.T) {
	blockchain := NewBlockchain()

	blockchain.AddBlock([]byte("Alice pays Bob 10"))
	blockchain.AddBlock([]byte("Bob pays Charlie 5"))

	blockchain.Blocks[1].Data = []byte("Alice pays Bob 10000")

	if blockchain.ValidateChain() {
		t.Fatal("expected tampered blockchain to fail validation")
	}
}
