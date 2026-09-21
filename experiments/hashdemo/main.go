package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

func main() {
	height := uint64(1)
	timestamp := int64(1000)
	data := []byte("ABD")
	prevHash := []byte("XYZ")

	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, height)
	fmt.Printf("After Height: %x\n", buf.Bytes())

	binary.Write(&buf, binary.BigEndian, timestamp)
	fmt.Printf("After Timestamp: %x\n", buf.Bytes())

	binary.Write(&buf, binary.BigEndian, uint64(len(data)))
	fmt.Printf("After Data Length: %x\n", buf.Bytes())

	buf.Write(data)
	fmt.Printf("After Data: %x\n", buf.Bytes())

	binary.Write(&buf, binary.BigEndian, uint64(len(prevHash)))
	fmt.Printf("After PrevHash Length: %x\n", buf.Bytes())

	buf.Write(prevHash)
	fmt.Printf("Final serialized bytes: %x\n", buf.Bytes())

	hash := sha256.Sum256(buf.Bytes())
	fmt.Printf("SHA-256: %x\n", hash)
}
