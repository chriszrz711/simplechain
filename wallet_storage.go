package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
)

type walletStorage struct {
	PrivateKey []byte `json:"private_key"`
}

type encryptedWalletStorage struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

const (
	walletSaltSize = 16
	walletKeySize  = 32

	walletArgonTime    uint32 = 2
	walletArgonMemory  uint32 = 32 * 1024
	walletArgonThreads uint8  = 2
)

func SaveWalletToFile(
	wallet *Wallet,
	path string,
) error {
	if wallet == nil {
		return fmt.Errorf("wallet cannot be nil")
	}

	if wallet.PrivateKey == nil {
		return fmt.Errorf("wallet private key cannot be nil")
	}

	// 把 ECDSA Private Key 编码成标准 DER 格式
	privateKeyDER, err := x509.MarshalECPrivateKey(
		wallet.PrivateKey,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to encode wallet private key: %w",
			err,
		)
	}

	data := walletStorage{
		PrivateKey: privateKeyDER,
	}

	encoded, err := json.MarshalIndent(
		data,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to encode wallet file: %w",
			err,
		)
	}

	// 和 blockchain 一样使用临时文件 + rename
	tmpPath := path + ".tmp"

	// Private Key 文件权限更严格：0600
	if err := os.WriteFile(
		tmpPath,
		encoded,
		0600,
	); err != nil {
		return fmt.Errorf(
			"failed to write temporary wallet file: %w",
			err,
		)
	}

	if err := os.Rename(
		tmpPath,
		path,
	); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf(
			"failed to replace wallet file: %w",
			err,
		)
	}

	return nil
}

func LoadWalletFromFile(
	path string,
) (*Wallet, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read wallet file: %w",
			err,
		)
	}

	var data walletStorage

	if err := json.Unmarshal(
		encoded,
		&data,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode wallet file: %w",
			err,
		)
	}

	if len(data.PrivateKey) == 0 {
		return nil, fmt.Errorf(
			"wallet file contains no private key",
		)
	}

	privateKey, err := x509.ParseECPrivateKey(
		data.PrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse wallet private key: %w",
			err,
		)
	}

	// Simplechain 当前固定使用 P-256
	if privateKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf(
			"wallet uses unsupported elliptic curve",
		)
	}
	// 从 Private Key 重新得到 Public Key
	publicKeyBytes, err := privateKey.PublicKey.Bytes()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to encode wallet public key: %w",
			err,
		)
	}

	wallet := &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKeyBytes,
	}
	return wallet, nil
}
func deriveWalletKey(
	password string,
	salt []byte,
) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		walletArgonTime,
		walletArgonMemory,
		walletArgonThreads,
		walletKeySize,
	)
}
func SaveWalletEncrypted(
	wallet *Wallet,
	path string,
	password string,
) error {
	if wallet == nil {
		return fmt.Errorf("wallet cannot be nil")
	}

	if wallet.PrivateKey == nil {
		return fmt.Errorf("wallet private key cannot be nil")
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// 1. Private Key → DER
	privateKeyDER, err := x509.MarshalECPrivateKey(
		wallet.PrivateKey,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to encode wallet private key: %w",
			err,
		)
	}

	// 2. 创建随机 salt
	salt := make([]byte, walletSaltSize)

	if _, err := io.ReadFull(
		rand.Reader,
		salt,
	); err != nil {
		return fmt.Errorf(
			"failed to generate wallet salt: %w",
			err,
		)
	}

	// 3. Password + Salt → 32-byte AES key
	key := deriveWalletKey(
		password,
		salt,
	)

	// 4. 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf(
			"failed to create AES cipher: %w",
			err,
		)
	}

	// 5. AES → GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf(
			"failed to create GCM cipher: %w",
			err,
		)
	}

	// 6. 每次加密生成新的随机 nonce
	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(
		rand.Reader,
		nonce,
	); err != nil {
		return fmt.Errorf(
			"failed to generate wallet nonce: %w",
			err,
		)
	}

	// 7. 真正加密 Private Key
	ciphertext := gcm.Seal(
		nil,
		nonce,
		privateKeyDER,
		nil,
	)

	// 8. 准备 JSON 数据
	data := encryptedWalletStorage{
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}

	encoded, err := json.MarshalIndent(
		data,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to encode encrypted wallet: %w",
			err,
		)
	}

	// 9. Atomic save
	tmpPath := path + ".tmp"

	if err := os.WriteFile(
		tmpPath,
		encoded,
		0600,
	); err != nil {
		return fmt.Errorf(
			"failed to write temporary encrypted wallet: %w",
			err,
		)
	}

	if err := os.Rename(
		tmpPath,
		path,
	); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf(
			"failed to replace encrypted wallet file: %w",
			err,
		)
	}

	return nil
}
func LoadWalletEncrypted(
	path string,
	password string,
) (*Wallet, error) {
	if password == "" {
		return nil, fmt.Errorf(
			"password cannot be empty",
		)
	}

	// 1. 读取文件
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read encrypted wallet file: %w",
			err,
		)
	}

	// 2. JSON decode
	var data encryptedWalletStorage

	if err := json.Unmarshal(
		encoded,
		&data,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode encrypted wallet file: %w",
			err,
		)
	}

	if len(data.Salt) == 0 ||
		len(data.Nonce) == 0 ||
		len(data.Ciphertext) == 0 {
		return nil, fmt.Errorf(
			"encrypted wallet file is incomplete",
		)
	}

	// 3. 用同样 Password + Salt 重新派生 AES key
	key := deriveWalletKey(
		password,
		data.Salt,
	)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create AES cipher: %w",
			err,
		)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create GCM cipher: %w",
			err,
		)
	}

	if len(data.Nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf(
			"invalid wallet nonce size",
		)
	}

	// 4. Ciphertext → Private Key DER
	privateKeyDER, err := gcm.Open(
		nil,
		data.Nonce,
		data.Ciphertext,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to decrypt wallet: wrong password or corrupted file",
		)
	}

	// 5. DER → ECDSA Private Key
	privateKey, err := x509.ParseECPrivateKey(
		privateKeyDER,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse decrypted private key: %w",
			err,
		)
	}

	if privateKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf(
			"wallet uses unsupported elliptic curve",
		)
	}

	// 6. Private Key → Public Key
	publicKeyBytes, err := privateKey.PublicKey.Bytes()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to encode wallet public key: %w",
			err,
		)
	}

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKeyBytes,
	}, nil
}
