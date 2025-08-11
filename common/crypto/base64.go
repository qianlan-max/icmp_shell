package crypto

import "encoding/base64"

type Base64Cryptor struct{}

func (c *Base64Cryptor) Encrypt(text []byte) ([]byte, error) {

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(text)))
	base64.StdEncoding.Encode(encoded, text)
	return encoded, nil
}

func (c *Base64Cryptor) Decrypt(text []byte) ([]byte, error) {
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(text)))
	n, err := base64.StdEncoding.Decode(decoded, text)
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}
