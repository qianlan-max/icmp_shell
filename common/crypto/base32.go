package crypto

import "encoding/base32"

type Base32Cryptor struct{}

func (c *Base32Cryptor) Encrypt(text []byte) ([]byte, error) {

	encoded := make([]byte, base32.StdEncoding.EncodedLen(len(text)))
	base32.StdEncoding.Encode(encoded, text)
	return encoded, nil
}

func (c *Base32Cryptor) Decrypt(text []byte) ([]byte, error) {
	decoded := make([]byte, base32.StdEncoding.DecodedLen(len(text)))
	n, err := base32.StdEncoding.Decode(decoded, text)
	if err != nil {
		return nil, err
	}

	return decoded[:n], nil
}
