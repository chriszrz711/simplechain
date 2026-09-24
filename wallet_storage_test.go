package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadWalletPreservesAddress(t *testing.T) {
	wallet := NewWallet()

	originalAddress := wallet.Address()

	path := filepath.Join(
		t.TempDir(),
		"wallet.json",
	)

	// 保存 Wallet
	if err := SaveWalletToFile(
		wallet,
		path,
	); err != nil {
		t.Fatalf(
			"failed to save wallet: %v",
			err,
		)
	}

	// 从硬盘重新加载
	loaded, err := LoadWalletFromFile(path)
	if err != nil {
		t.Fatalf(
			"failed to load wallet: %v",
			err,
		)
	}

	// 地址必须完全一样
	if loaded.Address() != originalAddress {
		t.Fatalf(
			"expected loaded address %s, got %s",
			originalAddress,
			loaded.Address(),
		)
	}
}
func TestLoadedWalletCanSpendExistingUTXO(t *testing.T) {
	// 1. 创建 Alice 和 Bob
	alice := NewWallet()
	bob := NewWallet()

	// 2. 创建 Blockchain
	bc := NewBlockchain()

	// 3. 给 Alice 50
	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf(
			"failed to add coinbase block: %v",
			err,
		)
	}

	// 4. 保存 Blockchain
	blockchainPath := filepath.Join(
		t.TempDir(),
		"blockchain.json",
	)

	if err := bc.SaveToFile(blockchainPath); err != nil {
		t.Fatalf(
			"failed to save blockchain: %v",
			err,
		)
	}

	// 5. 保存 Alice Wallet
	walletPath := filepath.Join(
		t.TempDir(),
		"alice_wallet.json",
	)

	if err := SaveWalletToFile(
		alice,
		walletPath,
	); err != nil {
		t.Fatalf(
			"failed to save Alice wallet: %v",
			err,
		)
	}

	// 6. 模拟程序重启：
	//    不再使用原来的 bc / alice
	loadedBC, err := LoadBlockchainFromFile(
		blockchainPath,
	)
	if err != nil {
		t.Fatalf(
			"failed to load blockchain: %v",
			err,
		)
	}

	loadedAlice, err := LoadWalletFromFile(
		walletPath,
	)
	if err != nil {
		t.Fatalf(
			"failed to load Alice wallet: %v",
			err,
		)
	}

	// 7. 加载后的 Alice 应该仍然有 50
	if loadedBC.GetBalance(
		loadedAlice.Address(),
	) != CoinbaseReward {
		t.Fatalf(
			"expected loaded Alice balance %d, got %d",
			CoinbaseReward,
			loadedBC.GetBalance(
				loadedAlice.Address(),
			),
		)
	}

	// 8. 加载后的 Alice 给 Bob 转 30
	payment, err := loadedBC.NewSignedTransaction(
		loadedAlice,
		bob.Address(),
		30,
	)
	if err != nil {
		t.Fatalf(
			"failed to create signed transaction after reload: %v",
			err,
		)
	}

	// 9. 把交易加入新区块
	if err := loadedBC.AddBlock([]Transaction{
		*payment,
	}); err != nil {
		t.Fatalf(
			"failed to add payment block after reload: %v",
			err,
		)
	}

	// 10. 检查余额
	if loadedBC.GetBalance(
		loadedAlice.Address(),
	) != 20 {
		t.Fatalf(
			"expected loaded Alice balance 20, got %d",
			loadedBC.GetBalance(
				loadedAlice.Address(),
			),
		)
	}

	if loadedBC.GetBalance(
		bob.Address(),
	) != 30 {
		t.Fatalf(
			"expected Bob balance 30, got %d",
			loadedBC.GetBalance(
				bob.Address(),
			),
		)
	}

	// 11. 整条链仍然合法
	if !loadedBC.ValidateChain() {
		t.Fatal(
			"expected blockchain to remain valid after reload and spend",
		)
	}
}
func TestLoadWalletFromMissingFileReturnsError(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"does-not-exist.json",
	)

	_, err := LoadWalletFromFile(path)

	if err == nil {
		t.Fatal(
			"expected error when loading missing wallet file",
		)
	}
}
func TestLoadWalletRejectsInvalidPrivateKey(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"wallet.json",
	)

	invalidWalletJSON := []byte(`{
		"private_key": "dGhpcyBpcyBub3QgYSByZWFsIHByaXZhdGUga2V5"
	}`)

	if err := os.WriteFile(
		path,
		invalidWalletJSON,
		0600,
	); err != nil {
		t.Fatalf(
			"failed to write invalid wallet file: %v",
			err,
		)
	}

	_, err := LoadWalletFromFile(path)

	if err == nil {
		t.Fatal(
			"expected invalid private key to be rejected",
		)
	}
}
func TestSaveAndLoadEncryptedWallet(t *testing.T) {
	wallet := NewWallet()

	originalAddress := wallet.Address()

	path := filepath.Join(
		t.TempDir(),
		"wallet.json",
	)

	password := "correct-horse-battery-staple"

	if err := SaveWalletEncrypted(
		wallet,
		path,
		password,
	); err != nil {
		t.Fatalf(
			"failed to save encrypted wallet: %v",
			err,
		)
	}

	loaded, err := LoadWalletEncrypted(
		path,
		password,
	)
	if err != nil {
		t.Fatalf(
			"failed to load encrypted wallet: %v",
			err,
		)
	}

	if loaded.Address() != originalAddress {
		t.Fatalf(
			"expected loaded address %s, got %s",
			originalAddress,
			loaded.Address(),
		)
	}
}
func TestLoadEncryptedWalletRejectsWrongPassword(t *testing.T) {
	wallet := NewWallet()

	path := filepath.Join(
		t.TempDir(),
		"wallet.json",
	)

	if err := SaveWalletEncrypted(
		wallet,
		path,
		"correct-password",
	); err != nil {
		t.Fatalf(
			"failed to save encrypted wallet: %v",
			err,
		)
	}

	_, err := LoadWalletEncrypted(
		path,
		"wrong-password",
	)

	if err == nil {
		t.Fatal(
			"expected wrong password to be rejected",
		)
	}
}
func TestEncryptedWalletUsesFreshSaltAndNonce(t *testing.T) {
	wallet := NewWallet()

	password := "same-password"

	dir := t.TempDir()

	path1 := filepath.Join(
		dir,
		"wallet1.json",
	)

	path2 := filepath.Join(
		dir,
		"wallet2.json",
	)

	// 同一个 Wallet + 同一个 Password 保存两次
	if err := SaveWalletEncrypted(
		wallet,
		path1,
		password,
	); err != nil {
		t.Fatalf(
			"failed to save first encrypted wallet: %v",
			err,
		)
	}

	if err := SaveWalletEncrypted(
		wallet,
		path2,
		password,
	); err != nil {
		t.Fatalf(
			"failed to save second encrypted wallet: %v",
			err,
		)
	}

	// 读取第一个文件
	encoded1, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf(
			"failed to read first wallet file: %v",
			err,
		)
	}

	var data1 encryptedWalletStorage

	if err := json.Unmarshal(
		encoded1,
		&data1,
	); err != nil {
		t.Fatalf(
			"failed to decode first wallet file: %v",
			err,
		)
	}

	// 读取第二个文件
	encoded2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf(
			"failed to read second wallet file: %v",
			err,
		)
	}

	var data2 encryptedWalletStorage

	if err := json.Unmarshal(
		encoded2,
		&data2,
	); err != nil {
		t.Fatalf(
			"failed to decode second wallet file: %v",
			err,
		)
	}

	// Salt 每次都应该重新随机生成
	if bytes.Equal(
		data1.Salt,
		data2.Salt,
	) {
		t.Fatal(
			"expected encrypted wallets to use different salts",
		)
	}

	// Nonce 每次也应该重新随机生成
	if bytes.Equal(
		data1.Nonce,
		data2.Nonce,
	) {
		t.Fatal(
			"expected encrypted wallets to use different nonces",
		)
	}

	// 最终 ciphertext 也应该不同
	if bytes.Equal(
		data1.Ciphertext,
		data2.Ciphertext,
	) {
		t.Fatal(
			"expected encrypted wallets to have different ciphertext",
		)
	}
}
func TestLoadEncryptedWalletRejectsTamperedCiphertext(t *testing.T) {
	wallet := NewWallet()

	password := "correct-password"

	path := filepath.Join(
		t.TempDir(),
		"wallet.json",
	)

	// 1. 正常保存加密 Wallet
	if err := SaveWalletEncrypted(
		wallet,
		path,
		password,
	); err != nil {
		t.Fatalf(
			"failed to save encrypted wallet: %v",
			err,
		)
	}

	// 2. 读取文件
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"failed to read encrypted wallet: %v",
			err,
		)
	}

	var data encryptedWalletStorage

	if err := json.Unmarshal(
		encoded,
		&data,
	); err != nil {
		t.Fatalf(
			"failed to decode encrypted wallet: %v",
			err,
		)
	}

	if len(data.Ciphertext) == 0 {
		t.Fatal(
			"expected ciphertext to be non-empty",
		)
	}

	// 3. 故意篡改 ciphertext 的一个 byte
	data.Ciphertext[0] ^= 0x01

	// 4. 写回文件
	tampered, err := json.MarshalIndent(
		data,
		"",
		"  ",
	)
	if err != nil {
		t.Fatalf(
			"failed to encode tampered wallet: %v",
			err,
		)
	}

	if err := os.WriteFile(
		path,
		tampered,
		0600,
	); err != nil {
		t.Fatalf(
			"failed to write tampered wallet: %v",
			err,
		)
	}

	// 5. 使用正确密码加载
	_, err = LoadWalletEncrypted(
		path,
		password,
	)

	if err == nil {
		t.Fatal(
			"expected tampered encrypted wallet to be rejected",
		)
	}
}
