package main

import (
	"fmt"
	"sort"
)

type Mempool struct {
	Transactions map[string]Transaction
}

func NewMempool() *Mempool {
	return &Mempool{
		Transactions: make(map[string]Transaction),
	}
}

// AddTransaction admits only transactions spending confirmed UTXOs.
func (mp *Mempool) AddTransaction(bc *Blockchain, tx *Transaction) error {
	if mp == nil {
		return fmt.Errorf("nil mempool")
	}
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}
	if tx.IsCoinbase() {
		return fmt.Errorf("coinbase transactions cannot enter the mempool")
	}
	if !tx.ValidateWithUTXOSet(bc.UTXOSet) {
		return fmt.Errorf("invalid transaction")
	}

	key := fmt.Sprintf("%x", tx.ID)
	if _, exists := mp.Transactions[key]; exists {
		return nil
	}

	reserved := make(map[string]bool)
	for _, pending := range mp.Transactions {
		for _, input := range pending.Inputs {
			reserved[fmt.Sprintf("%x:%d", input.TxID, input.OutIndex)] = true
		}
	}
	if !reserveMempoolInputs(tx, reserved) {
		return fmt.Errorf("transaction conflicts with a pending spend")
	}

	if mp.Transactions == nil {
		mp.Transactions = make(map[string]Transaction)
	}
	mp.Transactions[key] = cloneTransaction(*tx)
	return nil
}

func (mp *Mempool) Has(txID []byte) bool {
	if mp == nil {
		return false
	}
	_, exists := mp.Transactions[fmt.Sprintf("%x", txID)]
	return exists
}

func (mp *Mempool) Size() int {
	if mp == nil {
		return 0
	}
	return len(mp.Transactions)
}

// TransactionsForBlock returns independent snapshots in hex TxID order.
func (mp *Mempool) TransactionsForBlock() []Transaction {
	if mp == nil {
		return nil
	}
	keys := make([]string, 0, len(mp.Transactions))
	for key := range mp.Transactions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	transactions := make([]Transaction, 0, len(keys))
	for _, key := range keys {
		transactions = append(transactions, cloneTransaction(mp.Transactions[key]))
	}
	return transactions
}

func (mp *Mempool) RemoveTransactions(transactions []Transaction) {
	if mp == nil {
		return
	}
	for _, tx := range transactions {
		delete(mp.Transactions, fmt.Sprintf("%x", tx.ID))
	}
}

// Revalidate checks confirmed state only; pending outputs never become spendable.
func (mp *Mempool) Revalidate(bc *Blockchain) {
	if mp == nil || bc == nil {
		return
	}
	retained := make(map[string]Transaction)
	reserved := make(map[string]bool)
	for _, tx := range mp.TransactionsForBlock() {
		if tx.IsCoinbase() || !tx.ValidateWithUTXOSet(bc.UTXOSet) {
			continue
		}
		if !reserveMempoolInputs(&tx, reserved) {
			continue
		}
		retained[fmt.Sprintf("%x", tx.ID)] = tx
	}
	mp.Transactions = retained
}

// Reserve only after all inputs are known to be free.
func reserveMempoolInputs(tx *Transaction, reserved map[string]bool) bool {
	for _, input := range tx.Inputs {
		if reserved[fmt.Sprintf("%x:%d", input.TxID, input.OutIndex)] {
			return false
		}
	}
	for _, input := range tx.Inputs {
		reserved[fmt.Sprintf("%x:%d", input.TxID, input.OutIndex)] = true
	}
	return true
}
