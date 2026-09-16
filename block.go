package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

type Block struct {
	Height    uint64
	Timestamp int64
	Data      []byte
	PrevHash  []byte
	Hash      []byte
}
func NewBlock(height uint64, data []byte, prevHash []byte) *Block {
	block := &Block{
		Height:    height,
		Timestamp: time.Now().Unix(),
		Data:      data,
		PrevHash:  prevHash,
	}

	block.Hash = block.calculateHash()

	return block
}
func (b *Block) calculateHash() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, b.Height)
	binary.Write(&buf, binary.BigEndian, b.Timestamp)

	binary.Write(&buf, binary.BigEndian, uint64(len(b.Data)))
	buf.Write(b.Data)

	binary.Write(&buf, binary.BigEndian, uint64(len(b.PrevHash)))
	buf.Write(b.PrevHash)

	hash := sha256.Sum256(buf.Bytes())

	return hash[:]
}