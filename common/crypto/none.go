package crypto

// 明文通讯
type NoOpCryptor struct{}

func (c *NoOpCryptor) Encrypt(text []byte) ([]byte, error) {
	return text, nil
}

func (c *NoOpCryptor) Decrypt(text []byte) ([]byte, error) {
	return text, nil
}
