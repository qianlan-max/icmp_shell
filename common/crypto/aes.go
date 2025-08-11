package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

type AESCryptor struct {
	aead cipher.AEAD
}

// token处理后生成 AES-256 的 32 字节密钥。
func NewAESCryptor(token []byte) (*AESCryptor, error) {
	if len(token) == 0 {
		return nil, errors.New("AES 加密需要一个非空的 token/key")
	}

	key := sha256.Sum256(token)

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("创建 AES 加密块失败: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	return &AESCryptor{
		aead: aead,
	}, nil
}

func (c *AESCryptor) Encrypt(text []byte) ([]byte, error) {

	nonceSize := c.aead.NonceSize()
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成 Nonce 失败: %w", err)
	}

	return c.aead.Seal(nonce, nonce, text, nil), nil
}

func (c *AESCryptor) Decrypt(text []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	if len(text) < nonceSize {
		return nil, errors.New("密文太短")
	}

	nonce, ciphertext := text[:nonceSize], text[nonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密/认证失败: %w", err)
	}

	return plaintext, nil
}
