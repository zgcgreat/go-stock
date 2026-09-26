package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
)

// 凭据加解密（AES-GCM）。
// 密钥来源：环境变量 GO_STOCK_SECRET_KEY（推荐，32 字节）。
// 未配置时回退为内置开发密钥（仅本地开发，生产务必配置固定密钥）。
// 说明：加密用于避免明文凭据落入数据库/访问日志；密钥本身仍需通过环境变量安全管理。

const devFallbackKey = "go-stock-dev-insecure-secret-key-change-me!"

func getKey() []byte {
	if k := os.Getenv("GO_STOCK_SECRET_KEY"); k != "" {
		// 归一化到 32 字节
		b := []byte(k)
		if len(b) >= 32 {
			return b[:32]
		}
		padded := make([]byte, 32)
		copy(padded, b)
		return padded
	}
	log.Println("[WARN] GO_STOCK_SECRET_KEY 未设置，凭据使用内置开发密钥加密（生产环境不安全）。")
	return []byte(devFallbackKey)
}

// EncryptString 加密明文，返回 base64 字符串。
func EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(getKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString 解密 base64 密文，返回明文。
func DecryptString(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(getKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := data[:ns], data[ns:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
