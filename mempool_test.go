package main

import (
	"fmt"
	"testing"
)

func TestMempoolStoresTransactionByID(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()
	bob := NewWallet()

	// Alice 先获得 50
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

	// 创建一笔正常签名交易，但暂时不 AddBlock
	tx, err := bc.NewSignedTransaction(
		alice,
		bob.Address(),
		30,
	)
	if err != nil {
		t.Fatalf(
			"failed to create transaction: %v",
			err,
		)
	}

	mempool := NewMempool()

	mempool.Add(tx)

	key := fmt.Sprintf("%x", tx.ID)

	stored, exists := mempool.Transactions[key]

	if !exists {
		t.Fatal(
			"expected transaction to exist in mempool",
		)
	}

	if fmt.Sprintf("%x", stored.ID) !=
		fmt.Sprintf("%x", tx.ID) {

		t.Fatal(
			"stored transaction has wrong ID",
		)
	}
}
