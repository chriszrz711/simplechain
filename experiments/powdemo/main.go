package main

import (
	"crypto/sha256"
	"fmt"
	"math/big"
)

func main() {
	data := "Hello SimpleChain"

	targetBits := 20

	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	var nonce uint64 = 0

	for {
		input := fmt.Sprintf("%s%d", data, nonce)

		hash := sha256.Sum256([]byte(input))

		var hashInt big.Int
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(target) == -1 {
			fmt.Println("Mining successful!")
			fmt.Printf("Nonce: %d\n", nonce)
			fmt.Printf("Hash: %x\n", hash)
			break
		}

		nonce++
	}
}
