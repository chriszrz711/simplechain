package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

type Block struct {
	Height       uint64
	Timestamp    int64
	Transactions []Transaction
	PrevHash     []byte
	Hash         []byte
	Nonce        uint64
}

func NewBlock(height uint64, transactions []Transaction, prevHash []byte) *Block {
	block := cloneBlock(&Block{
		Height:       height,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PrevHash:     prevHash,
	})

	pow := NewProofOfWork(block)

	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash

	return block
}
func (b *Block) calculateHash() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, b.Height)
	binary.Write(&buf, binary.BigEndian, b.Timestamp)

	if len(b.Transactions) > 0 {
		txHash := b.hashTransactions()
		binary.Write(&buf, binary.BigEndian, uint64(len(txHash)))
		buf.Write(txHash)
	}
	binary.Write(&buf, binary.BigEndian, uint64(len(b.PrevHash)))
	buf.Write(b.PrevHash)
	binary.Write(&buf, binary.BigEndian, b.Nonce)

	hash := sha256.Sum256(buf.Bytes())

	return hash[:]
}
func (b *Block) hashTransactions() []byte {
	var txIDs [][]byte

	for _, tx := range b.Transactions {
		txIDs = append(txIDs, tx.ID)
	}

	joined := bytes.Join(txIDs, []byte{})

	hash := sha256.Sum256(joined)

	return hash[:]
}

const genesisTimestamp int64 = 1700000000

func NewGenesisBlock() *Block {
	block := &Block{
		Height:    0,
		Timestamp: genesisTimestamp,
		PrevHash:  []byte{},
	}

	pow := NewProofOfWork(block)

	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash

	return block
}
