package crypto

import "fmt"

func New(mode string, token []byte) (Cryptor, error) {
	switch mode {
	case "none":
		return &NoOpCryptor{}, nil
	case "xor":
		return NewXorCryptor(token)
	case "base64":
		return &Base64Cryptor{}, nil
	case "base32":
		return &Base32Cryptor{}, nil
	case "aes":
		return NewAESCryptor(token)
	default:
		return nil, fmt.Errorf("不支持的加密模式: %s", mode)
	}
}
