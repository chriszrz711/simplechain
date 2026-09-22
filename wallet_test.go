package main

import (
	"bytes"
	"testing"
)

func TestNewWalletCreatesKeyPair(t *testing.T) {
	wallet := NewWallet()

	if wallet.PrivateKey == nil {
		t.Fatal("expected private key to be created")
	}

	if len(wallet.PublicKey) == 0 {
		t.Fatal("expected public key to be created")
	}
}
func TestTwoWalletsHaveDifferentKeys(t *testing.T) {
	wallet1 := NewWallet()
	wallet2 := NewWallet()

	if wallet1.PrivateKey.Equal(wallet2.PrivateKey) {
		t.Fatal("expected two wallets to have different private keys")
	}

	if bytes.Equal(wallet1.PublicKey, wallet2.PublicKey) {
		t.Fatal("expected two wallets to have different public keys")
	}
}
func TestWalletCanSignAndVerifyData(t *testing.T) {
	wallet := NewWallet()

	data := []byte("Alice pays Bob 10")

	signature, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("failed to sign data: %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("expected signature to be created")
	}

	if !VerifySignature(wallet.PublicKey, data, signature) {
		t.Fatal("expected signature to be valid")
	}
}
func TestSignatureFailsIfDataChanges(t *testing.T) {
	wallet := NewWallet()

	data := []byte("Alice pays Bob 10")

	signature, err := wallet.Sign(data)
	if err != nil {
		t.Fatalf("failed to sign data: %v", err)
	}

	tamperedData := []byte("Alice pays Bob 1000")

	if VerifySignature(wallet.PublicKey, tamperedData, signature) {
		t.Fatal("signature should be invalid after data is changed")
	}
}
func TestSignatureFailsWithWrongPublicKey(t *testing.T) {
	aliceWallet := NewWallet()
	bobWallet := NewWallet()

	data := []byte("Alice pays Bob 10")

	signature, err := aliceWallet.Sign(data)
	if err != nil {
		t.Fatalf("failed to sign data: %v", err)
	}

	if VerifySignature(bobWallet.PublicKey, data, signature) {
		t.Fatal("signature should be invalid with the wrong public key")
	}
}
func TestWalletHasAddress(t *testing.T) {
	wallet := NewWallet()

	address := wallet.Address()

	if address == "" {
		t.Fatal("expected wallet address to be non-empty")
	}

	if wallet.Address() != address {
		t.Fatal("expected wallet address to be deterministic")
	}
}
