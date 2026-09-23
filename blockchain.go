package main

import (
	"bytes"
	"fmt"
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
func (bc *Blockchain) AddBlock(transactions []Transaction) error {
	// 1. 挖矿前先验证 transactions
	if !bc.ValidateTransactionsForNextBlock(transactions) {
		return fmt.Errorf("invalid transactions")
	}

	// 2. 根据当前链尾创建新区块
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	newBlock := NewBlock(
		lastBlock.Height+1,
		transactions,
		lastBlock.Hash,
	)

	// 3. Transactions 已经验证过了，
	//这里只检查新 Block 的结构和 PoW
	if !bc.validateNextBlockStructure(newBlock) {
		return fmt.Errorf("invalid block")
	}

	// 4. 全部通过后才真正修改 blockchain
	bc.Blocks = append(bc.Blocks, newBlock)

	return nil
}

// addBlockUnchecked constructs fixtures without validating the chain.
func (bc *Blockchain) addBlockUnchecked(transactions []Transaction) *Block {
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	newBlock := NewBlock(
		lastBlock.Height+1,
		transactions,
		lastBlock.Hash,
	)

	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

func (bc *Blockchain) ValidateChain() bool {
	if len(bc.Blocks) == 0 {
		return false
	}
	spentOutputs := make(map[string]bool)
	var previousTransactions []Transaction

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
		var valid bool
		previousTransactions, valid = validateTransactions(
			currentBlock.Transactions,
			previousTransactions,
			spentOutputs,
		)
		if !valid {
			return false
		}
	}

	return true
}
func (bc *Blockchain) AddBlockValidated(block *Block) error {
	// 先验证 candidate block
	if !bc.ValidateNextBlock(block) {
		return fmt.Errorf("invalid block")
	}

	// 验证通过以后才真正修改 blockchain
	bc.Blocks = append(bc.Blocks, block)

	return nil
}

func (bc *Blockchain) ValidateTransactionsForNextBlock(
	transactions []Transaction,
) bool {
	spentOutputs := make(map[string]bool)
	var previousTransactions []Transaction

	for _, block := range bc.Blocks {
		var valid bool
		previousTransactions, valid = validateTransactions(
			block.Transactions,
			previousTransactions,
			spentOutputs,
		)
		if !valid {
			return false
		}
	}

	_, valid := validateTransactions(transactions, previousTransactions, spentOutputs)
	return valid
}

func validateTransactions(
	transactions []Transaction,
	previousTransactions []Transaction,
	spentOutputs map[string]bool,
) ([]Transaction, bool) {
	coinbaseCount := 0
	for txIndex := range transactions {
		tx := &transactions[txIndex]

		if tx.IsCoinbase() {
			coinbaseCount++

			if coinbaseCount > 1 {
				return previousTransactions, false
			}

			if txIndex != 0 {
				return previousTransactions, false
			}
		}

		// 检查普通交易有没有重复花费同一个 Output
		if !tx.IsCoinbase() {
			seenInputs := make(map[string]bool)

			for _, input := range tx.Inputs {
				key := fmt.Sprintf(
					"%x:%d",
					input.TxID,
					input.OutIndex,
				)

				if spentOutputs[key] {
					return previousTransactions, false
				}

				if seenInputs[key] {
					return previousTransactions, false
				}

				seenInputs[key] = true
			}
		}

		if !tx.Validate(previousTransactions) {
			return previousTransactions, false
		}

		// Transaction 验证通过以后，
		// 才正式把它使用的 Outputs 标记为 spent
		if !tx.IsCoinbase() {
			for _, input := range tx.Inputs {
				key := fmt.Sprintf(
					"%x:%d",
					input.TxID,
					input.OutIndex,
				)

				spentOutputs[key] = true
			}
		}

		previousTransactions = append(
			previousTransactions,
			*tx,
		)
	}

	return previousTransactions, true
}
func (bc *Blockchain) ValidateNextBlock(block *Block) bool {
	// 第一层：Block 结构
	if !bc.validateNextBlockStructure(block) {
		return false
	}

	// 第二层：Block 中的 Transactions
	if !bc.ValidateTransactionsForNextBlock(block.Transactions) {
		return false
	}

	return true
}
func (bc *Blockchain) validateNextBlockStructure(block *Block) bool {
	if block == nil {
		return false
	}

	if len(bc.Blocks) == 0 {
		return false
	}

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	// Height 必须连续
	if block.Height != lastBlock.Height+1 {
		return false
	}

	// 必须连接当前链尾
	if !bytes.Equal(block.PrevHash, lastBlock.Hash) {
		return false
	}

	// PoW 必须有效
	pow := NewProofOfWork(block)

	if !pow.Validate() {
		return false
	}

	return true
}
