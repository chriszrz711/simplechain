package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type blockchainStorage struct {
	Blocks []*Block `json:"blocks"`
}

func (bc *Blockchain) SaveToFile(path string) error {
	if bc == nil {
		return fmt.Errorf("blockchain cannot be nil")
	}

	if !bc.ValidateChain() {
		return fmt.Errorf("cannot save invalid blockchain")
	}

	data := blockchainStorage{
		Blocks: bc.Blocks,
	}

	encoded, err := json.MarshalIndent(
		data,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to encode blockchain: %w",
			err,
		)
	}

	tmpPath := path + ".tmp"

	if err := os.WriteFile(
		tmpPath,
		encoded,
		0644,
	); err != nil {
		return fmt.Errorf(
			"failed to write temporary blockchain file: %w",
			err,
		)
	}

	if err := os.Rename(
		tmpPath,
		path,
	); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf(
			"failed to replace blockchain file: %w",
			err,
		)
	}

	return nil
}
func LoadBlockchainFromFile(
	path string,
) (*Blockchain, error) {

	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read blockchain file: %w",
			err,
		)
	}

	var data blockchainStorage

	if err := json.Unmarshal(
		encoded,
		&data,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode blockchain: %w",
			err,
		)
	}

	if len(data.Blocks) == 0 {
		return nil, fmt.Errorf(
			"blockchain file contains no blocks",
		)
	}

	bc := &Blockchain{
		Blocks: data.Blocks,
	}

	// 不信任硬盘里的内容
	if !bc.ValidateChain() {
		return nil, fmt.Errorf(
			"loaded blockchain is invalid",
		)
	}

	// Blocks 是 source of truth
	// UTXOSet 从历史重新构建
	bc.UTXOSet = bc.BuildUTXOSet()

	return bc, nil
}
