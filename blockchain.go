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
		coinbaseCount := 0
		for txIndex := range currentBlock.Transactions {
			tx := &currentBlock.Transactions[txIndex]

			if tx.IsCoinbase() {
				coinbaseCount++

				if coinbaseCount > 1 {
					return false
				}

				if txIndex != 0 {
					return false
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
						return false
					}

					if seenInputs[key] {
						return false
					}

					seenInputs[key] = true
				}
			}

			if !tx.Validate(previousTransactions) {
				return false
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
	}

	return true
}
func (bc *Blockchain) AddBlockValidated(block *Block) error {
	// 1. 临时把新区块加入链
	bc.Blocks = append(bc.Blocks, block)

	// 2. 验证整条链
	if !bc.ValidateChain() {

		// 3. 验证失败：撤销刚才的 append
		bc.Blocks = bc.Blocks[:len(bc.Blocks)-1]

		return fmt.Errorf("invalid block")
	}

	// 4. 验证成功：保留新区块
	return nil
}
