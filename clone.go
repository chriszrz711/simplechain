package main

import "bytes"

func cloneTransaction(tx Transaction) Transaction {
	tx.ID = bytes.Clone(tx.ID)
	tx.Inputs = append([]TXInput(nil), tx.Inputs...)
	for i := range tx.Inputs {
		tx.Inputs[i].TxID = bytes.Clone(tx.Inputs[i].TxID)
		tx.Inputs[i].Signature = bytes.Clone(tx.Inputs[i].Signature)
		tx.Inputs[i].PublicKey = bytes.Clone(tx.Inputs[i].PublicKey)
	}
	tx.Outputs = append([]TXOutput(nil), tx.Outputs...)
	return tx
}

func cloneBlock(block *Block) *Block {
	if block == nil {
		return nil
	}
	cloned := *block
	cloned.Hash = bytes.Clone(block.Hash)
	cloned.PrevHash = bytes.Clone(block.PrevHash)
	if block.Transactions != nil {
		cloned.Transactions = make([]Transaction, len(block.Transactions))
		for i, tx := range block.Transactions {
			cloned.Transactions[i] = cloneTransaction(tx)
		}
	}
	return &cloned
}
