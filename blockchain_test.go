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

	tx1 := NewCoinbaseTransaction("Alice", CoinbaseReward)
	tx2 := NewCoinbaseTransaction("Bob", CoinbaseReward)
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
func TestAddBlockWithMultipleTransactions(t *testing.T) {
	blockchain := NewBlockchain()

	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Bob", "Charlie", 5)
	tx3 := NewTransaction("Charlie", "Dave", 2)

	blockchain.AddBlock([]Transaction{
		*tx1,
		*tx2,
		*tx3,
	})

	if len(blockchain.Blocks) != 2 {
		t.Fatalf(
			"expected blockchain to contain 2 blocks, got %d",
			len(blockchain.Blocks),
		)
	}

	block := blockchain.Blocks[1]

	if len(block.Transactions) != 3 {
		t.Fatalf(
			"expected block to contain 3 transactions, got %d",
			len(block.Transactions),
		)
	}

	if block.Transactions[0].Amount != 10 {
		t.Fatalf(
			"expected first transaction amount 10, got %d",
			block.Transactions[0].Amount,
		)
	}

	if block.Transactions[1].Amount != 5 {
		t.Fatalf(
			"expected second transaction amount 5, got %d",
			block.Transactions[1].Amount,
		)
	}

	if block.Transactions[2].Amount != 2 {
		t.Fatalf(
			"expected third transaction amount 2, got %d",
			block.Transactions[2].Amount,
		)
	}
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
func TestValidateChainRejectsIncorrectBlockHeight(t *testing.T) {
	blockchain := NewBlockchain()

	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Bob", "Charlie", 5)

	blockchain.AddBlock([]Transaction{*tx1})
	blockchain.AddBlock([]Transaction{*tx2})

	block2 := blockchain.Blocks[2]

	// 篡改高度
	block2.Height = 99

	// 重新挖矿，让这个区块自己的 PoW 再次合法
	pow := NewProofOfWork(block2)
	nonce, hash := pow.Run()

	block2.Nonce = nonce
	block2.Hash = hash

	// 即使 PoW 合法，错误的 Height 仍然应该让整条链无效
	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject incorrect block height")
	}
}
func TestValidateChainRejectsGenesisWithNonEmptyPrevHash(t *testing.T) {
	blockchain := NewBlockchain()

	genesis := blockchain.Blocks[0]

	// 篡改 Genesis 的 PrevHash
	genesis.PrevHash = []byte("fake previous hash")

	// 重新挖矿，让 Genesis 自己的 PoW 再次合法
	pow := NewProofOfWork(genesis)
	nonce, hash := pow.Run()

	genesis.Nonce = nonce
	genesis.Hash = hash

	// 即使 PoW 合法，Genesis 也不能有 PrevHash
	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject genesis block with non-empty previous hash")
	}
}
func TestValidateChainRejectsEmptyBlockchain(t *testing.T) {
	blockchain := &Blockchain{}

	if blockchain.ValidateChain() {
		t.Fatal("empty blockchain should be invalid")
	}
}

func TestValidateChainRejectsInvalidTransactionIDAfterRemining(t *testing.T) {
	blockchain := NewBlockchain()
	tx := NewCoinbaseTransaction("Alice", CoinbaseReward)
	blockchain.AddBlock([]Transaction{*tx})

	if !blockchain.ValidateChain() {
		t.Fatal("blockchain should be valid before transaction tampering")
	}

	block := blockchain.Blocks[1]

	// 故意不调用 SetID()，让交易内容与原 ID 不一致。
	block.Transactions[0].To = "Mallory"
	block.Transactions[0].Outputs[0].To = "Mallory"

	if !block.Transactions[0].ValidateCoinbase() {
		t.Fatal("tampered transaction should retain valid coinbase structure")
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash

	if !pow.Validate() {
		t.Fatal("remined block should have valid proof of work")
	}

	if block.Transactions[0].ValidateID() {
		t.Fatal("tampered transaction should still have an invalid ID after remining")
	}

	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject an invalid transaction ID even after remining")
	}
}

func TestValidateChainAcceptsValidSignedTransaction(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	blockchain := NewBlockchain()

	// Block 1：Coinbase 创建 50 给 Alice
	funding := NewCoinbaseTransaction(
		aliceWallet.Address(),
		CoinbaseReward,
	)

	blockchain.AddBlock([]Transaction{
		*funding,
	})

	// Block 2：Alice 花自己的 50
	// 10 给 Bob，40 找零给自己
	spend := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    bobWallet.Address(),
			},
			{
				Value: 40,
				To:    aliceWallet.Address(),
			},
		},
	}

	spend.SetID()

	if err := spend.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	blockchain.AddBlock([]Transaction{
		spend,
	})

	if !blockchain.ValidateChain() {
		t.Fatal("expected blockchain with valid signed transaction to be valid")
	}
}
func TestValidateChainRejectsMultipleCoinbaseTransactionsInOneBlock(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	blockchain := NewBlockchain()

	coinbase1 := NewCoinbaseTransaction(
		aliceWallet.Address(),
		CoinbaseReward,
	)

	coinbase2 := NewCoinbaseTransaction(
		bobWallet.Address(),
		CoinbaseReward,
	)

	blockchain.AddBlock([]Transaction{
		*coinbase1,
		*coinbase2,
	})

	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject multiple coinbase transactions in one block")
	}
}
func TestValidateChainRejectsCoinbaseNotFirstInBlock(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()
	minerWallet := NewWallet()

	blockchain := NewBlockchain()

	// Block 1:
	// Coinbase 给 Alice 50
	funding := NewCoinbaseTransaction(
		aliceWallet.Address(),
		CoinbaseReward,
	)

	blockchain.AddBlock([]Transaction{
		*funding,
	})

	// Alice 花 funding:0
	spend := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 10,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 10,
				To:    bobWallet.Address(),
			},
			{
				Value: 40,
				To:    aliceWallet.Address(),
			},
		},
	}

	spend.SetID()

	if err := spend.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign spend transaction: %v", err)
	}

	// Block 2 的 Coinbase
	coinbase := NewCoinbaseTransaction(
		minerWallet.Address(),
		CoinbaseReward,
	)

	// 故意把普通交易放前面，Coinbase 放第二
	blockchain.AddBlock([]Transaction{
		spend,
		*coinbase,
	})

	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject coinbase transaction that is not first in block")
	}
}
func TestValidateChainRejectsDoubleSpend(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()
	charlieWallet := NewWallet()

	blockchain := NewBlockchain()

	// Block 1:
	// Alice 获得 50
	funding := NewCoinbaseTransaction(
		aliceWallet.Address(),
		CoinbaseReward,
	)

	blockchain.AddBlock([]Transaction{
		*funding,
	})

	// Block 2:
	// Alice 第一次花 funding:0
	spend1 := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 50,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    bobWallet.Address(),
			},
		},
	}

	spend1.SetID()

	if err := spend1.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign first spend: %v", err)
	}

	blockchain.AddBlock([]Transaction{
		spend1,
	})

	// Block 3:
	// Alice 再次花完全相同的 funding:0
	spend2 := Transaction{
		From:   aliceWallet.Address(),
		To:     charlieWallet.Address(),
		Amount: 50,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    charlieWallet.Address(),
			},
		},
	}

	spend2.SetID()

	if err := spend2.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign second spend: %v", err)
	}

	blockchain.AddBlock([]Transaction{
		spend2,
	})

	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject double spending of the same output")
	}
}
func TestValidateChainRejectsDuplicateInputInSameTransaction(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	blockchain := NewBlockchain()

	funding := NewCoinbaseTransaction(
		aliceWallet.Address(),
		CoinbaseReward,
	)

	blockchain.AddBlock([]Transaction{
		*funding,
	})

	spend := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 50,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 50,
				To:    bobWallet.Address(),
			},
		},
	}

	spend.SetID()

	if err := spend.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	blockchain.AddBlock([]Transaction{
		spend,
	})

	if blockchain.ValidateChain() {
		t.Fatal("blockchain should reject duplicate use of the same output within one transaction")
	}
}
func TestTransactionRejectsNegativeOutput(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	funding := NewCoinbaseTransaction(
		aliceWallet.Address(),
		CoinbaseReward,
	)

	spend := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 100,
		Inputs: []TXInput{
			{
				TxID:     funding.ID,
				OutIndex: 0,
				From:     aliceWallet.Address(),
			},
		},
		Outputs: []TXOutput{
			{
				Value: 100,
				To:    bobWallet.Address(),
			},
			{
				Value: -50,
				To:    aliceWallet.Address(),
			},
		},
	}

	spend.SetID()

	if err := spend.Sign(aliceWallet); err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	if spend.Validate([]Transaction{*funding}) {
		t.Fatal("transaction should reject negative output values")
	}
}
func TestTransactionRejectsNonCoinbaseWithoutInputs(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	tx := Transaction{
		From:   aliceWallet.Address(),
		To:     bobWallet.Address(),
		Amount: 10,
	}

	tx.SetID()

	if tx.Validate(nil) {
		t.Fatal("non-coinbase transaction without inputs should be rejected")
	}
}

func TestAddBlockValidatedAcceptsValidBlock(t *testing.T) {
	bc := NewBlockchain()

	originalLength := len(bc.Blocks)

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	validBlock := NewBlock(
		lastBlock.Height+1,
		[]Transaction{},
		lastBlock.Hash,
	)

	err := bc.AddBlockValidated(validBlock)

	if err != nil {
		t.Fatalf(
			"expected valid block to be accepted, got error: %v",
			err,
		)
	}

	if len(bc.Blocks) != originalLength+1 {
		t.Fatalf(
			"expected blockchain length %d, got %d",
			originalLength+1,
			len(bc.Blocks),
		)
	}
}
