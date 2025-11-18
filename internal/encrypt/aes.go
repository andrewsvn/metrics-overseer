package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

const (
	aesKeySize = 32
)

type aesEncrypted struct {
	Key        []byte
	Nonce      []byte
	Ciphertext []byte
}

func generateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func aesEncrypt(data []byte) (*aesEncrypted, error) {
	enc := &aesEncrypted{}
	var err error

	enc.Key, err = generateRandomBytes(aesKeySize)
	if err != nil {
		return nil, err
	}

	aesBlock, err := aes.NewCipher(enc.Key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, err
	}

	enc.Nonce, err = generateRandomBytes(aesGCM.NonceSize())
	if err != nil {
		return nil, err
	}

	enc.Ciphertext = aesGCM.Seal(nil, enc.Nonce, data, nil)
	return enc, nil
}

func aesDecrypt(enc *aesEncrypted) ([]byte, error) {
	aesBlock, err := aes.NewCipher(enc.Key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, err
	}

	data, err := aesGCM.Open(nil, enc.Nonce, enc.Ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}
