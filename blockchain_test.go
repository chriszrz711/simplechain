package main

import (
	"bytes"
	"fmt"
	"maps"
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
	blockchain.addBlockUnchecked([]Transaction{*tx})

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
	blockchain.addBlockUnchecked([]Transaction{*tx1})
	blockchain.addBlockUnchecked([]Transaction{*tx2})

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
	if err := blockchain.AddBlock([]Transaction{*tx1}); err != nil {
		t.Fatalf("failed to add valid block: %v", err)
	}
	if err := blockchain.AddBlock([]Transaction{*tx2}); err != nil {
		t.Fatalf("failed to add valid block: %v", err)
	}

	if !blockchain.ValidateChain() {
		t.Fatal("expected valid blockchain to pass validation")
	}
}
func TestValidateChainDetectsTamperedTransactions(t *testing.T) {
	blockchain := NewBlockchain()

	tx1 := NewTransaction("Alice", "Bob", 10)
	tx2 := NewTransaction("Bob", "Charlie", 5)
	blockchain.addBlockUnchecked([]Transaction{*tx1})
	blockchain.addBlockUnchecked([]Transaction{*tx2})

	blockchain.Blocks[1].Transactions[0].Amount = 10000
	blockchain.Blocks[1].Transactions[0].SetID()

	if blockchain.ValidateChain() {
		t.Fatal("expected tampered blockchain to fail validation")
	}
}
func TestValidateChainDetectsTamperedGenesisBlock(t *testing.T) {
	blockchain := NewBlockchain()
	tx := NewTransaction("Alice", "Bob", 10)
	blockchain.addBlockUnchecked([]Transaction{*tx})

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

	blockchain.addBlockUnchecked([]Transaction{
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

	blockchain.addBlockUnchecked([]Transaction{*tx1})
	blockchain.addBlockUnchecked([]Transaction{*tx2})

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
	if err := blockchain.AddBlock([]Transaction{*tx}); err != nil {
		t.Fatalf("failed to add valid block: %v", err)
	}

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

	if err := blockchain.AddBlock([]Transaction{
		*funding,
	}); err != nil {
		t.Fatalf("failed to add valid block: %v", err)
	}

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

	if err := blockchain.AddBlock([]Transaction{
		spend,
	}); err != nil {
		t.Fatalf("failed to add valid block: %v", err)
	}

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

	blockchain.addBlockUnchecked([]Transaction{
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

	blockchain.addBlockUnchecked([]Transaction{
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
	blockchain.addBlockUnchecked([]Transaction{
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

	blockchain.addBlockUnchecked([]Transaction{
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

	blockchain.addBlockUnchecked([]Transaction{
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

	blockchain.addBlockUnchecked([]Transaction{
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

	blockchain.addBlockUnchecked([]Transaction{
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

	blockchain.addBlockUnchecked([]Transaction{
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

func TestAddBlockRejectsInvalidTransactions(t *testing.T) {
	bc := NewBlockchain()
	originalLength := len(bc.Blocks)
	originalTip := bc.Blocks[originalLength-1]
	tx := NewTransaction("Alice", "Bob", 10)

	if err := bc.AddBlock([]Transaction{*tx}); err == nil {
		t.Fatal("expected invalid transactions to be rejected")
	}

	if len(bc.Blocks) != originalLength {
		t.Fatal("rejected transaction must not change blockchain length")
	}
	if bc.Blocks[originalLength-1] != originalTip {
		t.Fatal("rejected transaction must not replace the existing tip")
	}
	if !bc.ValidateChain() {
		t.Fatal("blockchain should remain valid after rejection")
	}
}

func TestAddBlockAcceptsValidTransactions(t *testing.T) {
	bc := NewBlockchain()
	originalLength := len(bc.Blocks)
	tx := NewCoinbaseTransaction("Alice", CoinbaseReward)

	if err := bc.AddBlock([]Transaction{*tx}); err != nil {
		t.Fatalf("failed to add valid block: %v", err)
	}

	if len(bc.Blocks) != originalLength+1 {
		t.Fatal("valid transaction should add exactly one block")
	}
	block := bc.Blocks[originalLength]
	if len(block.Transactions) != 1 || !bytes.Equal(block.Transactions[0].ID, tx.ID) {
		t.Fatal("added block should contain the supplied transaction")
	}
	if !bc.ValidateChain() {
		t.Fatal("blockchain should remain valid after adding a valid transaction")
	}
}
func TestValidateNextBlockAcceptsValidBlock(t *testing.T) {
	bc := NewBlockchain()

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	candidate := NewBlock(
		lastBlock.Height+1,
		[]Transaction{},
		lastBlock.Hash,
	)

	if !bc.ValidateNextBlock(candidate) {
		t.Fatal("expected valid next block to be accepted")
	}
}
func TestValidateNextBlockRejectsWrongHeight(t *testing.T) {
	bc := NewBlockchain()

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	candidate := NewBlock(
		lastBlock.Height+1,
		[]Transaction{},
		lastBlock.Hash,
	)

	// 故意篡改 Height
	candidate.Height = lastBlock.Height + 2

	if bc.ValidateNextBlock(candidate) {
		t.Fatal("expected block with wrong height to be rejected")
	}
}
func TestValidateNextBlockRejectsWrongPrevHash(t *testing.T) {
	bc := NewBlockchain()

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	candidate := NewBlock(
		lastBlock.Height+1,
		[]Transaction{},
		lastBlock.Hash,
	)

	// 故意让它指向错误的前一个 Hash
	candidate.PrevHash = []byte("wrong previous hash")

	if bc.ValidateNextBlock(candidate) {
		t.Fatal("expected block with wrong previous hash to be rejected")
	}
}
func TestValidateNextBlockRejectsInvalidPoW(t *testing.T) {
	bc := NewBlockchain()

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	candidate := NewBlock(
		lastBlock.Height+1,
		[]Transaction{},
		lastBlock.Hash,
	)

	// 不断修改 Nonce，直到确认 PoW 无效
	for {
		candidate.Nonce++

		pow := NewProofOfWork(candidate)

		if !pow.Validate() {
			break
		}
	}

	if bc.ValidateNextBlock(candidate) {
		t.Fatal("expected block with invalid proof of work to be rejected")
	}
}
func TestBuildUTXOSetIncludesUnspentCoinbaseOutput(t *testing.T) {
	bc := NewBlockchain()

	coinbase := NewCoinbaseTransaction(
		"Alice",
		CoinbaseReward,
	)

	err := bc.AddBlock([]Transaction{
		*coinbase,
	})
	if err != nil {
		t.Fatalf("failed to add block: %v", err)
	}

	utxoSet := bc.BuildUTXOSet()

	key := fmt.Sprintf(
		"%x:%d",
		coinbase.ID,
		0,
	)

	output, exists := utxoSet[key]

	if !exists {
		t.Fatal("expected coinbase output to exist in UTXO set")
	}

	if output.Value != CoinbaseReward {
		t.Fatalf(
			"expected UTXO value %d, got %d",
			CoinbaseReward,
			output.Value,
		)
	}

	if output.To != "Alice" {
		t.Fatalf(
			"expected UTXO owner Alice, got %s",
			output.To,
		)
	}
}
func TestBuildUTXOSetRemovesSpentOutput(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()
	bob := NewWallet()

	// 1. Alice 先通过 Coinbase 获得 50
	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf("failed to add coinbase block: %v", err)
	}

	// 2. Alice 花 30 给 Bob
	payment, err := bc.NewSignedTransaction(
		alice,
		bob.Address(),
		30,
	)
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	if err := bc.AddBlock([]Transaction{
		*payment,
	}); err != nil {
		t.Fatalf("failed to add payment block: %v", err)
	}

	// 3. 根据整条链重新构建当前 UTXO Set
	utxoSet := bc.BuildUTXOSet()

	// Coinbase 原来的 50 已经被 Alice 花掉
	oldKey := fmt.Sprintf(
		"%x:%d",
		coinbase.ID,
		0,
	)

	if _, exists := utxoSet[oldKey]; exists {
		t.Fatal("spent coinbase output should not remain in UTXO set")
	}

	// payment output 0 = Bob 30
	bobKey := fmt.Sprintf(
		"%x:%d",
		payment.ID,
		0,
	)

	bobOutput, exists := utxoSet[bobKey]
	if !exists {
		t.Fatal("expected Bob payment output in UTXO set")
	}

	if bobOutput.Value != 30 {
		t.Fatalf(
			"expected Bob UTXO value 30, got %d",
			bobOutput.Value,
		)
	}

	if bobOutput.To != bob.Address() {
		t.Fatal("expected payment output to belong to Bob")
	}

	// payment output 1 = Alice 找零 20
	aliceChangeKey := fmt.Sprintf(
		"%x:%d",
		payment.ID,
		1,
	)

	aliceChange, exists := utxoSet[aliceChangeKey]
	if !exists {
		t.Fatal("expected Alice change output in UTXO set")
	}

	if aliceChange.Value != 20 {
		t.Fatalf(
			"expected Alice change value 20, got %d",
			aliceChange.Value,
		)
	}

	if aliceChange.To != alice.Address() {
		t.Fatal("expected change output to belong to Alice")
	}
}
func TestBlockchainMaintainsUTXOSet(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()

	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf("failed to add coinbase block: %v", err)
	}

	key := fmt.Sprintf(
		"%x:%d",
		coinbase.ID,
		0,
	)

	output, exists := bc.UTXOSet[key]

	if !exists {
		t.Fatal(
			"expected coinbase output to exist in blockchain UTXO set",
		)
	}

	if output.Value != CoinbaseReward {
		t.Fatalf(
			"expected UTXO value %d, got %d",
			CoinbaseReward,
			output.Value,
		)
	}

	if output.To != alice.Address() {
		t.Fatal(
			"expected UTXO to belong to Alice",
		)
	}
}
func TestBlockchainUTXOSetUpdatesAfterSpend(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()
	bob := NewWallet()

	// 1. Alice 通过 Coinbase 获得 50
	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf(
			"failed to add coinbase block: %v",
			err,
		)
	}

	// 2. Alice 花 30 给 Bob
	payment, err := bc.NewSignedTransaction(
		alice,
		bob.Address(),
		30,
	)
	if err != nil {
		t.Fatalf(
			"failed to create payment transaction: %v",
			err,
		)
	}

	if err := bc.AddBlock([]Transaction{
		*payment,
	}); err != nil {
		t.Fatalf(
			"failed to add payment block: %v",
			err,
		)
	}

	// 3. 原来的 Coinbase UTXO 必须已经消失
	oldKey := fmt.Sprintf(
		"%x:%d",
		coinbase.ID,
		0,
	)

	if _, exists := bc.UTXOSet[oldKey]; exists {
		t.Fatal(
			"spent coinbase output should not remain in blockchain UTXO set",
		)
	}

	// 4. Bob 应该有新的 30
	bobKey := fmt.Sprintf(
		"%x:%d",
		payment.ID,
		0,
	)

	bobOutput, exists := bc.UTXOSet[bobKey]
	if !exists {
		t.Fatal(
			"expected Bob payment output in blockchain UTXO set",
		)
	}

	if bobOutput.Value != 30 {
		t.Fatalf(
			"expected Bob UTXO value 30, got %d",
			bobOutput.Value,
		)
	}

	if bobOutput.To != bob.Address() {
		t.Fatal(
			"expected payment output to belong to Bob",
		)
	}

	// 5. Alice 应该有 20 找零
	aliceChangeKey := fmt.Sprintf(
		"%x:%d",
		payment.ID,
		1,
	)

	aliceChange, exists := bc.UTXOSet[aliceChangeKey]
	if !exists {
		t.Fatal(
			"expected Alice change output in blockchain UTXO set",
		)
	}

	if aliceChange.Value != 20 {
		t.Fatalf(
			"expected Alice change value 20, got %d",
			aliceChange.Value,
		)
	}

	if aliceChange.To != alice.Address() {
		t.Fatal(
			"expected change output to belong to Alice",
		)
	}
}
func TestBlockchainUTXOSetMatchesRebuiltUTXOSet(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()
	bob := NewWallet()
	charlie := NewWallet()

	// 1. Alice 获得 Coinbase 50
	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf(
			"failed to add coinbase block: %v",
			err,
		)
	}

	// 2. Alice 给 Bob 30，Alice 找零 20
	payment1, err := bc.NewSignedTransaction(
		alice,
		bob.Address(),
		30,
	)
	if err != nil {
		t.Fatalf(
			"failed to create first payment: %v",
			err,
		)
	}

	if err := bc.AddBlock([]Transaction{
		*payment1,
	}); err != nil {
		t.Fatalf(
			"failed to add first payment block: %v",
			err,
		)
	}

	// 3. Bob 再给 Charlie 10
	payment2, err := bc.NewSignedTransaction(
		bob,
		charlie.Address(),
		10,
	)
	if err != nil {
		t.Fatalf(
			"failed to create second payment: %v",
			err,
		)
	}

	if err := bc.AddBlock([]Transaction{
		*payment2,
	}); err != nil {
		t.Fatalf(
			"failed to add second payment block: %v",
			err,
		)
	}

	// 4. 从完整 Blockchain 历史重新计算
	rebuilt := bc.BuildUTXOSet()

	// 5. 数量必须一样
	if len(bc.UTXOSet) != len(rebuilt) {
		t.Fatalf(
			"UTXO set size mismatch: cached=%d rebuilt=%d",
			len(bc.UTXOSet),
			len(rebuilt),
		)
	}

	// 6. 每一个 cached UTXO 都必须和 rebuilt 完全一样
	for key, cachedOutput := range bc.UTXOSet {
		rebuiltOutput, exists := rebuilt[key]

		if !exists {
			t.Fatalf(
				"cached UTXO %s missing from rebuilt UTXO set",
				key,
			)
		}

		if cachedOutput != rebuiltOutput {
			t.Fatalf(
				"UTXO mismatch for %s: cached=%+v rebuilt=%+v",
				key,
				cachedOutput,
				rebuiltOutput,
			)
		}
	}
}

func TestValidateTransactionsForNextBlockWithUTXOSet(t *testing.T) {
	for _, name := range []string{
		"success_preserves_utxo_set",
		"failure_preserves_utxo_set",
		"rejects_candidate_double_spend",
		"accepts_same_block_dependent_spend",
		"uses_maintained_utxo_set",
	} {
		t.Run(name, func(t *testing.T) {
			bc := NewBlockchain()
			alice := NewWallet()
			bob := NewWallet()
			charlie := NewWallet()
			funding := NewCoinbaseTransaction(alice.Address(), CoinbaseReward)
			if err := bc.AddBlock([]Transaction{*funding}); err != nil {
				t.Fatalf("failed to add funding: %v", err)
			}

			payment, err := bc.NewSignedTransaction(alice, bob.Address(), 30)
			if err != nil {
				t.Fatalf("failed to create payment: %v", err)
			}
			if !payment.ValidateWithUTXOSet(bc.UTXOSet) {
				t.Fatal("initial payment must be valid")
			}
			candidates := []Transaction{*payment}
			wantValid := true

			switch name {
			case "failure_preserves_utxo_set":
				// Fail after the first valid transaction has changed temporary state.
				invalid := NewTransaction("Alice", "Bob", 10)
				candidates = append(candidates, *invalid)
				wantValid = false
			case "rejects_candidate_double_spend":
				second, err := bc.NewSignedTransaction(alice, charlie.Address(), 20)
				if err != nil {
					t.Fatalf("failed to create second spend: %v", err)
				}
				if !second.ValidateWithUTXOSet(bc.UTXOSet) {
					t.Fatal("second spend must be valid on its own")
				}
				candidates = append(candidates, *second)
				wantValid = false
			case "accepts_same_block_dependent_spend":
				candidateSet := cloneUTXOSet(bc.UTXOSet)
				applyTransactionsToUTXOSet(candidateSet, []Transaction{*payment})
				second, err := NewSignedUTXOTransactionFromSet(bob, charlie.Address(), 10, candidateSet)
				if err != nil {
					t.Fatalf("failed to create dependent spend: %v", err)
				}
				if second.ValidateWithUTXOSet(bc.UTXOSet) {
					t.Fatal("dependent spend must require the first candidate's output")
				}
				candidates = append(candidates, *second)
			case "uses_maintained_utxo_set":
				// History still contains funding, but next-block validation must use the map.
				delete(bc.UTXOSet, fmt.Sprintf("%x:%d", funding.ID, 0))
				wantValid = false
			}

			before := maps.Clone(bc.UTXOSet)
			originalMap := bc.UTXOSet
			if got := bc.ValidateTransactionsForNextBlock(candidates); got != wantValid {
				t.Errorf("expected candidate validation %t, got %t", wantValid, got)
			}
			if !maps.Equal(bc.UTXOSet, before) || !maps.Equal(originalMap, before) {
				t.Fatal("candidate validation must not mutate the maintained UTXO set")
			}
		})
	}
}
