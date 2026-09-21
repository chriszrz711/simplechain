package main

import "fmt"

func main() {
	tx := NewTransaction("Alice", "Bob", 10)
	block := NewBlock(
		1,
		[]Transaction{*tx},
		[]byte("previous hash"),
	)

	fmt.Printf("Height: %d\n", block.Height)
	fmt.Printf("Timestamp: %d\n", block.Timestamp)
	fmt.Printf("Transactions: %+v\n", block.Transactions)
	fmt.Printf("PrevHash: %x\n", block.PrevHash)
	fmt.Printf("Nonce: %d\n", block.Nonce)
	fmt.Printf("Hash: %x\n", block.Hash)
}
