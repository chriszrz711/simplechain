package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

const CoinbaseReward = 50

type Transaction struct {
	ID       []byte
	From     string
	To       string
	Amount   int
	Coinbase bool
	Inputs   []TXInput
	Outputs  []TXOutput
}
type TXOutput struct {
	Value int
	To    string
}
type TXInput struct {
	TxID      []byte
	OutIndex  int
	From      string
	Signature []byte
	PublicKey []byte
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
func (out *TXOutput) CanBeUnlockedWith(publicKey []byte) bool {
	return out.To == PublicKeyToAddress(publicKey)
}

func (tx *Transaction) VerifyInputs(transactions []Transaction) bool {
	for _, input := range tx.Inputs {
		found := false

		for _, previousTx := range transactions {
			if !bytes.Equal(input.TxID, previousTx.ID) {
				continue
			}

			if input.OutIndex < 0 || input.OutIndex >= len(previousTx.Outputs) {
				return false
			}

			previousOutput := previousTx.Outputs[input.OutIndex]

			if !previousOutput.CanBeUnlockedWith(input.PublicKey) {
				return false
			}

			found = true
			break
		}

		if !found {
			return false
		}
	}

	return true
}
func (tx *Transaction) SigningData() []byte {
	var data bytes.Buffer

	fmt.Fprintf(
		&data,
		"%s|%s|%d|%t|",
		tx.From,
		tx.To,
		tx.Amount,
		tx.Coinbase,
	)

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

	return data.Bytes()
}

func (tx *Transaction) SetID() {
	hash := sha256.Sum256(tx.SigningData())

	tx.ID = hash[:]
}
func (tx *Transaction) Sign(wallet *Wallet) error {
	signature, err := wallet.Sign(tx.SigningData())
	if err != nil {
		return err
	}

	for i := range tx.Inputs {
		tx.Inputs[i].Signature = signature
		tx.Inputs[i].PublicKey = wallet.PublicKey
	}

	return nil
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

	if amount <= 0 {
		return nil, fmt.Errorf(
			"amount must be positive",
		)
	}
	if to == "" {
		return nil, fmt.Errorf(
			"recipient cannot be empty",
		)
	}

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

func NewSignedUTXOTransaction(
	wallet *Wallet,
	to string,
	amount int,
	transactions []Transaction,
) (*Transaction, error) {

	if wallet == nil {
		return nil, fmt.Errorf("wallet cannot be nil")
	}

	tx, err := NewUTXOTransaction(
		wallet.Address(),
		to,
		amount,
		transactions,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Sign(wallet); err != nil {
		return nil, err
	}

	return tx, nil
}

func (tx *Transaction) ValidatePaymentDetails() bool {
	if tx.To == "" {
		return false
	}

	if tx.Amount <= 0 {
		return false
	}

	if len(tx.Outputs) == 0 {
		return false
	}

	paymentOutput := tx.Outputs[0]

	if paymentOutput.To != tx.To {
		return false
	}

	if paymentOutput.Value != tx.Amount {
		return false
	}

	return true
}
func (tx *Transaction) VerifySignatures() bool {
	data := tx.SigningData()

	for _, input := range tx.Inputs {
		if !VerifySignature(
			input.PublicKey,
			data,
			input.Signature,
		) {
			return false
		}
	}

	return true
}
func (tx *Transaction) IsCoinbase() bool {
	return tx.Coinbase
}
func (tx *Transaction) ValidateSenderDetails() bool {
	if tx.From == "" {
		return false
	}

	for _, input := range tx.Inputs {
		if input.From != tx.From {
			return false
		}

		if len(input.PublicKey) == 0 {
			return false
		}

		if PublicKeyToAddress(input.PublicKey) != tx.From {
			return false
		}
	}

	return true
}

func (tx *Transaction) ValidateAmounts(previousTransactions []Transaction) bool {
	inputTotal := 0
	seenInputs := make(map[string]bool)

	for _, input := range tx.Inputs {
		key := fmt.Sprintf(
			"%x:%d",
			input.TxID,
			input.OutIndex,
		)

		// 同一笔交易不能把同一个 UTXO 算两次
		if seenInputs[key] {
			return false
		}
		seenInputs[key] = true

		found := false

		for _, previousTx := range previousTransactions {
			if !bytes.Equal(input.TxID, previousTx.ID) {
				continue
			}

			if input.OutIndex < 0 ||
				input.OutIndex >= len(previousTx.Outputs) {
				return false
			}

			inputTotal += previousTx.Outputs[input.OutIndex].Value
			found = true
			break
		}

		if !found {
			return false
		}
	}

	outputTotal := 0

	for _, output := range tx.Outputs {
		if output.Value <= 0 {
			return false
		}
		if output.To == "" {
			return false
		}

		outputTotal += output.Value
	}

	return inputTotal == outputTotal
}

func (tx *Transaction) ValidateCoinbase() bool {
	if !tx.IsCoinbase() {
		return false
	}
	if tx.To == "" {
		return false
	}
	if len(tx.Inputs) != 0 {
		return false
	}

	if tx.Amount != CoinbaseReward {
		return false
	}

	if len(tx.Outputs) != 1 {
		return false
	}

	if tx.Outputs[0].Value != CoinbaseReward {
		return false
	}

	if tx.Outputs[0].To != tx.To {
		return false
	}

	return true
}
func (tx *Transaction) Validate(previousTransactions []Transaction) bool {
	if !tx.ValidateID() {
		return false
	}

	if tx.IsCoinbase() {
		return tx.ValidateCoinbase()
	}

	if len(tx.Inputs) == 0 {
		return false
	}

	if !tx.ValidateSenderDetails() {
		return false
	}

	if !tx.ValidatePaymentDetails() {
		return false
	}

	if !tx.VerifySignatures() {
		return false
	}

	if !tx.VerifyInputs(previousTransactions) {
		return false
	}

	if !tx.ValidateAmounts(previousTransactions) {
		return false
	}

	return true
}
func NewCoinbaseTransaction(to string, amount int) *Transaction {
	tx := &Transaction{
		To:       to,
		Amount:   amount,
		Coinbase: true,
		Outputs: []TXOutput{
			{
				Value: amount,
				To:    to,
			},
		},
	}

	tx.SetID()

	return tx
}
