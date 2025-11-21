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
	errEncryptingDisabled = errors.New("encrypting disabled")
	errDecryptingDisabled = errors.New("decrypting disabled")
)
