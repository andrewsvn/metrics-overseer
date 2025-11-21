package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
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
		return nil, fmt.Errorf("error generating AES key bytes: %w", err)
	}

	aesBlock, err := aes.NewCipher(enc.Key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher %w", err)
	}

	aesGCM, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("error creating AES GCM: %w", err)
	}

	enc.Nonce, err = generateRandomBytes(aesGCM.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("error generating AES nonce bytes: %w", err)
	}

	enc.Ciphertext = aesGCM.Seal(nil, enc.Nonce, data, nil)
	return enc, nil
}

func aesDecrypt(enc *aesEncrypted) ([]byte, error) {
	aesBlock, err := aes.NewCipher(enc.Key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("error creating AES GCM: %w", err)
	}

	data, err := aesGCM.Open(nil, enc.Nonce, enc.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("error decrypting AES ciphertext: %w", err)
	}
	return data, nil
}
