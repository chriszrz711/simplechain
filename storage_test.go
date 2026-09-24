package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadBlockchain(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()

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

	// 使用测试专用临时目录
	path := filepath.Join(
		t.TempDir(),
		"blockchain.json",
	)

	// 保存
	if err := bc.SaveToFile(path); err != nil {
		t.Fatalf(
			"failed to save blockchain: %v",
			err,
		)
	}

	// 重新加载
	loaded, err := LoadBlockchainFromFile(path)
	if err != nil {
		t.Fatalf(
			"failed to load blockchain: %v",
			err,
		)
	}

	// Block 数量应该完全一样
	if len(loaded.Blocks) != len(bc.Blocks) {
		t.Fatalf(
			"expected %d blocks, got %d",
			len(bc.Blocks),
			len(loaded.Blocks),
		)
	}

	// 加载后的链仍然必须合法
	if !loaded.ValidateChain() {
		t.Fatal(
			"expected loaded blockchain to be valid",
		)
	}

	// UTXOSet 应该从 Blocks 正确恢复
	if loaded.GetBalance(alice.Address()) != CoinbaseReward {
		t.Fatalf(
			"expected Alice balance %d, got %d",
			CoinbaseReward,
			loaded.GetBalance(alice.Address()),
		)
	}
}
func TestLoadBlockchainFromMissingFileReturnsError(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"does-not-exist.json",
	)

	_, err := LoadBlockchainFromFile(path)

	if err == nil {
		t.Fatal(
			"expected error when loading missing blockchain file",
		)
	}
}
func TestLoadBlockchainRejectsTamperedFile(t *testing.T) {
	bc := NewBlockchain()

	alice := NewWallet()

	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf(
			"failed to add block: %v",
			err,
		)
	}

	path := filepath.Join(
		t.TempDir(),
		"blockchain.json",
	)

	// 1. 先正常保存
	if err := bc.SaveToFile(path); err != nil {
		t.Fatalf(
			"failed to save blockchain: %v",
			err,
		)
	}

	// 2. 从硬盘读取 JSON
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"failed to read saved blockchain: %v",
			err,
		)
	}

	var data blockchainStorage

	if err := json.Unmarshal(
		encoded,
		&data,
	); err != nil {
		t.Fatalf(
			"failed to decode saved blockchain: %v",
			err,
		)
	}

	// 3. 人为篡改第二个 Block
	data.Blocks[1].Height++

	// 4. 把篡改后的数据重新写回硬盘
	tampered, err := json.MarshalIndent(
		data,
		"",
		"  ",
	)
	if err != nil {
		t.Fatalf(
			"failed to encode tampered blockchain: %v",
			err,
		)
	}

	if err := os.WriteFile(
		path,
		tampered,
		0644,
	); err != nil {
		t.Fatalf(
			"failed to write tampered blockchain: %v",
			err,
		)
	}

	// 5. 再尝试加载
	_, err = LoadBlockchainFromFile(path)

	if err == nil {
		t.Fatal(
			"expected tampered blockchain file to be rejected",
		)
	}
}
func TestSaveToFileReplacesExistingBlockchain(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"blockchain.json",
	)

	bc := NewBlockchain()

	alice := NewWallet()

	coinbase := NewCoinbaseTransaction(
		alice.Address(),
		CoinbaseReward,
	)

	// 第一次保存：只有 Genesis
	if err := bc.SaveToFile(path); err != nil {
		t.Fatalf(
			"failed to save initial blockchain: %v",
			err,
		)
	}

	// 添加新区块
	if err := bc.AddBlock([]Transaction{
		*coinbase,
	}); err != nil {
		t.Fatalf(
			"failed to add block: %v",
			err,
		)
	}

	// 第二次保存：应该替换旧文件
	if err := bc.SaveToFile(path); err != nil {
		t.Fatalf(
			"failed to replace blockchain file: %v",
			err,
		)
	}

	loaded, err := LoadBlockchainFromFile(path)
	if err != nil {
		t.Fatalf(
			"failed to load blockchain: %v",
			err,
		)
	}

	if len(loaded.Blocks) != 2 {
		t.Fatalf(
			"expected 2 blocks after replacement, got %d",
			len(loaded.Blocks),
		)
	}

	if loaded.GetBalance(alice.Address()) != CoinbaseReward {
		t.Fatalf(
			"expected Alice balance %d, got %d",
			CoinbaseReward,
			loaded.GetBalance(alice.Address()),
		)
	}
}
