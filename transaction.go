package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

type Transaction struct {
	ID      []byte
	From    string
	To      string
	Amount  int
	Inputs  []TXInput
	Outputs []TXOutput
}
type TXOutput struct {
	Value int
	To    string
}
type TXInput struct {
	TxID     []byte
	OutIndex int
	From     string
}
type UTXO struct {
	TxID     []byte
	OutIndex int
	Output   TXOutput
}

func NewTXOutput(value int, to string) *TXOutput {
	return &TXOutput{
		Value: value,
		To:    to,
	}
}
func NewTXInput(txID []byte, outIndex int, from string) *TXInput {
	return &TXInput{
		TxID:     txID,
		OutIndex: outIndex,
		From:     from,
	}
}
func NewTransaction(from string, to string, amount int) *Transaction {
	tx := &Transaction{
		From:   from,
		To:     to,
		Amount: amount,
	}

	tx.SetID()

	return tx
}
func (tx *Transaction) SetID() {
	var data bytes.Buffer

	fmt.Fprintf(&data, "%s|%s|%d|", tx.From, tx.To, tx.Amount)

	for _, in := range tx.Inputs {
		fmt.Fprintf(
			&data,
			"%x:%d:%s|",
			in.TxID,
			in.OutIndex,
			in.From,
		)
	}

	fmt.Fprint(&data, "->|")

	for _, out := range tx.Outputs {
		fmt.Fprintf(
			&data,
			"%d:%s|",
			out.Value,
			out.To,
		)
	}

	hash := sha256.Sum256(data.Bytes())

	tx.ID = hash[:]
}
func (tx *Transaction) ValidateID() bool {
	originalID := tx.ID

	txCopy := *tx
	txCopy.SetID()

	return bytes.Equal(originalID, txCopy.ID)
}

func FindUTXO(transactions []Transaction, owner string) []UTXO {
	spent := make(map[string]bool)

	for _, tx := range transactions {
		for _, in := range tx.Inputs {
			key := fmt.Sprintf("%x:%d", in.TxID, in.OutIndex)
			spent[key] = true
		}
	}

	var utxos []UTXO

	for _, tx := range transactions {
		for index, out := range tx.Outputs {
			key := fmt.Sprintf("%x:%d", tx.ID, index)

			if out.To == owner && !spent[key] {
				utxos = append(utxos, UTXO{
					TxID:     tx.ID,
					OutIndex: index,
					Output:   out,
				})
			}
		}
	}

	return utxos
}
func GetBalance(transactions []Transaction, owner string) int {
	utxos := FindUTXO(transactions, owner)

	balance := 0

	for _, utxo := range utxos {
		balance += utxo.Output.Value
	}

	return balance
}
func FindSpendableUTXO(
	transactions []Transaction,
	owner string,
	amount int,
) (int, []UTXO) {
	utxos := FindUTXO(transactions, owner)

	total := 0
	var selected []UTXO

	for _, utxo := range utxos {
		selected = append(selected, utxo)
		total += utxo.Output.Value

		if total >= amount {
			break
		}
	}

	return total, selected
}
func NewUTXOTransaction(
	from string,
	to string,
	amount int,
	transactions []Transaction,
) (*Transaction, error) {

	total, selected := FindSpendableUTXO(
		transactions,
		from,
		amount,
	)

	if total < amount {
		return nil, fmt.Errorf(
			"insufficient funds: have %d, need %d",
			total,
			amount,
		)
	}

	var inputs []TXInput

	for _, utxo := range selected {
		inputs = append(inputs, TXInput{
			TxID:     utxo.TxID,
			OutIndex: utxo.OutIndex,
			From:     from,
		})
	}

	outputs := []TXOutput{
		{
			Value: amount,
			To:    to,
		},
	}

	if total > amount {
		outputs = append(outputs, TXOutput{
			Value: total - amount,
			To:    from,
		})
	}

	tx := &Transaction{
		From:    from,
		To:      to,
		Amount:  amount,
		Inputs:  inputs,
		Outputs: outputs,
	}

	tx.SetID()

	return tx, nil
}
