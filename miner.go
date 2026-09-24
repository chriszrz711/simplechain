package main

import "fmt"

// MineBlock mines pending transactions through the safe local AddBlock path.
func (bc *Blockchain) MineBlock(mempool *Mempool, minerAddress string) (*Block, error) {
	if bc == nil {
		return nil, fmt.Errorf("nil blockchain")
	}
	if mempool == nil {
		return nil, fmt.Errorf("nil mempool")
	}
	if minerAddress == "" {
		return nil, fmt.Errorf("miner address cannot be empty")
	}
	if len(bc.Blocks) == 0 || bc.Blocks[len(bc.Blocks)-1] == nil {
		return nil, fmt.Errorf("blockchain must have a tip")
	}
	if bc.UTXOSet == nil {
		return nil, fmt.Errorf("blockchain UTXO set is not initialized")
	}

	mempool.Revalidate(bc)
	pending := mempool.TransactionsForBlock()
	coinbase := NewCoinbaseTransaction(minerAddress, CoinbaseReward)
	transactions := make([]Transaction, 0, len(pending)+1)
	transactions = append(transactions, *coinbase)
	transactions = append(transactions, pending...)

	if err := bc.AddBlock(transactions); err != nil {
		return nil, err
	}

	block := bc.Blocks[len(bc.Blocks)-1]
	mempool.RemoveTransactions(pending)
	mempool.Revalidate(bc)
	return block, nil
}
