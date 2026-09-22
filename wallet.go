package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() *Wallet {
	privateKey, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
	if err != nil {
		panic(err)
	}

	publicKey, err := privateKey.PublicKey.Bytes()
	if err != nil {
		panic(err)
	}

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
}
func (w *Wallet) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)

	signature, err := ecdsa.SignASN1(
		rand.Reader,
		w.PrivateKey,
		hash[:],
	)
	if err != nil {
		return nil, err
	}

	return signature, nil
}
func VerifySignature(publicKeyBytes []byte, data []byte, signature []byte) bool {
	publicKey, err := ecdsa.ParseUncompressedPublicKey(
		elliptic.P256(),
		publicKeyBytes,
	)
	if err != nil {
		return false
	}

	hash := sha256.Sum256(data)

	return ecdsa.VerifyASN1(
		publicKey,
		hash[:],
		signature,
	)
}
func PublicKeyToAddress(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)

	return fmt.Sprintf("%x", hash[:20])
}
func (w *Wallet) Address() string {
	return PublicKeyToAddress(w.PublicKey)
}
