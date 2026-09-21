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

	if len(genesis.Transactions) != 0 {
		t.Fatalf(
			"expected genesis transactions to be empty, got %d",
			len(genesis.Transactions),
		)
	}
}
func TestAddBlockLinksToPreviousBlock(t *testing.T) {
	blockchain := NewBlockchain()

	tx := NewTransaction("Alice", "Bob", 10)
	blockchain.AddBlock([]Transaction{*tx})

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
func TestAddMultipleBlocksLinksCorrectly(t *testing.T) {
	blockchain := NewBlockchain()

	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Bob", "Charlie", 5)
	blockchain.AddBlock([]Transaction{*tx1})
	blockchain.AddBlock([]Transaction{*tx2})

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

	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Bob", "Charlie", 5)
	blockchain.AddBlock([]Transaction{*tx1})
	blockchain.AddBlock([]Transaction{*tx2})

	if !blockchain.ValidateChain() {
		t.Fatal("expected valid blockchain to pass validation")
	}
}
func TestValidateChainDetectsTamperedTransactions(t *testing.T) {
	blockchain := NewBlockchain()

	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Bob", "Charlie", 5)
	blockchain.AddBlock([]Transaction{*tx1})
	blockchain.AddBlock([]Transaction{*tx2})

	blockchain.Blocks[1].Transactions[0].Amount = 10000
	blockchain.Blocks[1].Transactions[0].SetID()

	if blockchain.ValidateChain() {
		t.Fatal("expected tampered blockchain to fail validation")
	}
}
func TestValidateChainDetectsTamperedGenesisBlock(t *testing.T) {
	blockchain := NewBlockchain()
	tx := NewTransaction("Alice", "Bob", 10)
	blockchain.AddBlock([]Transaction{*tx})

	tamperedTx := NewTransaction("Alice", "Bob", 10000)
	blockchain.Blocks[0].Transactions = []Transaction{*tamperedTx}

	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject a tampered genesis block")
	}
}
func TestGenesisBlockHasFixedTimestamp(t *testing.T) {
	genesis := NewGenesisBlock()

	const expectedTimestamp int64 = 1700000000

	if genesis.Timestamp != expectedTimestamp {
		t.Fatalf(
			"expected genesis timestamp %d, got %d",
			expectedTimestamp,
			genesis.Timestamp,
		)
	}
}
func TestNewBlockchainUsesFixedGenesisBlock(t *testing.T) {
	blockchain := NewBlockchain()

	genesis := blockchain.Blocks[0]

	if genesis.Timestamp != genesisTimestamp {
		t.Fatalf(
			"expected blockchain genesis timestamp %d, got %d",
			genesisTimestamp,
			genesis.Timestamp,
		)
	}
}
