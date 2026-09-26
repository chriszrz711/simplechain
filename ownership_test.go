package main

import (
	"bytes"
	"encoding/json"
	"maps"
	"testing"
)

func ownershipSnapshot(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mutateCallerTransaction(tx *Transaction) {
	tx.ID[0] ^= 0xff
	tx.From = "changed"
	tx.To = "changed"
	tx.Amount++
	tx.Coinbase = !tx.Coinbase
	tx.CoinbaseHeight++
	tx.Outputs[0].Value = 999999
	tx.Outputs[0].To = "changed"
	if len(tx.Inputs) > 0 {
		tx.Inputs[0].TxID[0] ^= 0xff
		tx.Inputs[0].Signature[0] ^= 0xff
		tx.Inputs[0].PublicKey[0] ^= 0xff
		tx.Inputs[0].OutIndex++
		tx.Inputs[0].From = "changed"
	}
}

func TestNewBlockCopiesCallerOwnedData(t *testing.T) {
	prevHash := []byte{1, 2, 3}
	transactions := []Transaction{{
		ID: []byte{4, 5}, From: "Alice", To: "Bob", Amount: 10,
		Inputs:  []TXInput{{TxID: []byte{6}, Signature: []byte{7}, PublicKey: []byte{8}, From: "Alice"}},
		Outputs: []TXOutput{{Value: 10, To: "Bob"}},
	}}
	before := ownershipSnapshot(t, transactions)
	block := NewBlock(1, transactions, prevHash)
	if !bytes.Equal(before, ownershipSnapshot(t, transactions)) {
		t.Fatal("NewBlock modified caller transactions")
	}
	expected := ownershipSnapshot(t, block)
	prevHash[0] ^= 0xff
	mutateCallerTransaction(&transactions[0])
	if !bytes.Equal(expected, ownershipSnapshot(t, block)) {
		t.Error("NewBlock retained caller-owned backing data")
	}
	if !NewProofOfWork(block).Validate() {
		t.Error("caller mutation invalidated block PoW")
	}
}

func TestAddBlockValidatedStoresIndependentCopy(t *testing.T) {
	for _, signed := range []bool{false, true} {
		name := "coinbase"
		if signed {
			name = "signed inputs"
		}
		t.Run(name, func(t *testing.T) {
			bc := NewBlockchain()
			miner := NewWallet()
			tx := NewCoinbaseTransactionForHeight(miner.Address(), CoinbaseReward, 1)
			height := uint64(1)
			if signed {
				if err := bc.AddBlock([]Transaction{*tx}); err != nil {
					t.Fatal(err)
				}
				var err error
				tx, err = bc.NewSignedTransaction(miner, "Bob", 10)
				if err != nil {
					t.Fatal(err)
				}
				height = 2
			}
			// Separate the fixture from the existing chain even before NewBlock is fixed.
			prevHash := bytes.Clone(bc.Blocks[len(bc.Blocks)-1].Hash)
			candidate := NewBlock(height, []Transaction{*tx}, prevHash)
			expected := ownershipSnapshot(t, candidate)
			if err := bc.AddBlockValidated(candidate); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(expected, ownershipSnapshot(t, candidate)) {
				t.Fatal("admission modified caller block")
			}
			balance := bc.GetBalance(miner.Address())
			candidate.Height = 999
			candidate.Timestamp++
			candidate.Nonce++
			candidate.Hash[0] ^= 0xff
			candidate.PrevHash[0] ^= 0xff
			mutateCallerTransaction(&candidate.Transactions[0])
			accepted := bc.Blocks[len(bc.Blocks)-1]
			if accepted == candidate {
				t.Error("accepted block aliases caller pointer")
			}
			if !bytes.Equal(expected, ownershipSnapshot(t, accepted)) {
				t.Error("accepted block changed with caller data")
			}
			if !bc.ValidateChain() {
				t.Error("caller mutation invalidated chain")
			}
			if bc.GetBalance(miner.Address()) != balance {
				t.Error("caller mutation changed balance")
			}
			if !signed && balance != CoinbaseReward {
				t.Error("expected coinbase reward")
			}
			if !maps.Equal(bc.UTXOSet, bc.BuildUTXOSet()) {
				t.Error("cached UTXOs differ from history")
			}
		})
	}
}

func TestAddBlockDoesNotRetainCallerTransactionData(t *testing.T) {
	bc := NewBlockchain()
	miner := "Miner"
	transactions := []Transaction{*NewCoinbaseTransactionForHeight(miner, CoinbaseReward, 1)}
	before := ownershipSnapshot(t, transactions)
	if err := bc.AddBlock(transactions); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, ownershipSnapshot(t, transactions)) {
		t.Fatal("AddBlock modified caller transactions")
	}
	expected := ownershipSnapshot(t, bc.Blocks[1])
	mutateCallerTransaction(&transactions[0])
	if !bytes.Equal(expected, ownershipSnapshot(t, bc.Blocks[1])) {
		t.Error("AddBlock retained caller transaction data")
	}
	if !bc.ValidateChain() {
		t.Error("caller mutation invalidated chain")
	}
	if bc.GetBalance(miner) != CoinbaseReward {
		t.Error("unexpected miner balance")
	}
	if !maps.Equal(bc.UTXOSet, bc.BuildUTXOSet()) {
		t.Error("cached UTXOs differ from history")
	}
}
