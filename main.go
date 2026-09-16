package main

import "fmt"

func main() {
	block := NewBlock(
		1,
		[]byte("Alice pays Bob 10"),
		[]byte("previous hash"),
	)

	fmt.Printf("Height: %d\n", block.Height)
	fmt.Printf("Timestamp: %d\n", block.Timestamp)
	fmt.Printf("Data: %s\n", block.Data)
	fmt.Printf("PrevHash: %x\n", block.PrevHash)
	fmt.Printf("Hash: %x\n", block.Hash)
}
