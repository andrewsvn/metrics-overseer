package encrypt

import "errors"

type Encrypter interface {
	Encrypt([]byte) ([]byte, error)
	EncryptingEnabled() bool
}

type Decrypter interface {
	Decrypt([]byte) ([]byte, error)
	DecryptingEnabled() bool
}

var (
	ErrEncryptingDisabled = errors.New("encrypting disabled")
	ErrDecryptingDisabled = errors.New("decrypting disabled")
)
