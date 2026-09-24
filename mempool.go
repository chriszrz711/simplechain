package main

import "fmt"

type Mempool struct {
	Transactions map[string]Transaction
}

func NewMempool() *Mempool {
	return &Mempool{
		Transactions: make(map[string]Transaction),
	}
}

func (mp *Mempool) Add(tx *Transaction) {
	if tx == nil {
		return
	}

	key := fmt.Sprintf("%x", tx.ID)

	mp.Transactions[key] = *tx
}
