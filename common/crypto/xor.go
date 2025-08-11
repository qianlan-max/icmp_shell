package crypto

import (
	"crypto/md5"
	"errors"
)

type XorCryptor struct {
	token    []byte
	tokenMD5 []byte
}

func NewXorCryptor(token []byte) (*XorCryptor, error) {
	if token == nil || len(token) == 0 {
		return nil, errors.New("XOR加密需要一个非空的Token")
	}
	return &XorCryptor{
		token: token,
	}, nil
}

// Token生成MD5，参与
func (c *XorCryptor) getTokenMD5() ([]byte, error) {
	if c.token == nil {
		return nil, errors.New("auth: token is empty")
	}
	if c.tokenMD5 == nil {
		md5Handle := md5.New()
		md5Handle.Write(c.token)
		c.tokenMD5 = md5Handle.Sum(nil)
	}
	return c.tokenMD5, nil
}

func (c *XorCryptor) Encrypt(text []byte) ([]byte, error) {
	key, err := c.getTokenMD5()
	if err != nil {
		return nil, err
	}
	encryptedText := make([]byte, len(text))

	for i := 0; i < len(text); i++ {
		tempByte := text[i]
		for j := 0; j < 4; j++ {
			tempByte = tempByte ^ key[j]
		}
		encryptedText[i] = tempByte
	}
	return encryptedText, nil
}

func (c *XorCryptor) Decrypt(text []byte) ([]byte, error) {
	key, err := c.getTokenMD5()
	if err != nil {
		return nil, err
	}

	decryptedText := make([]byte, len(text))

	for i := 0; i < len(text); i++ {
		tempByte := text[i]
		for j := 3; j >= 0; j-- {
			tempByte = tempByte ^ key[j]
		}
		decryptedText[i] = tempByte
	}
	return decryptedText, nil
}
