package encrypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
)

const (
	publicKeyBlockType  = "PUBLIC KEY"
	privateKeyBlockType = "RSA PRIVATE KEY"
)

// RSAAESEngine allows encryption/decryption of byte data using combination of RSA+AES algorithms.
// For actual payload encryption AES algorithm is used, AES key is encrypted with RSA public key and decrypted with a
// corresponding private key.
// Engine implements both Encrypter and Decrypter interfaces, but each of them uses only one key, so if only
// Encrypter is needed then RSAAESEngine can be built only with a public key provided, similar story for Decrypter
// RSAAESEngine instances must be constructed using RSAEngineBuilder
type RSAAESEngine struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
	label      []byte
}

// rsaAESEncryptedBlock is a serializable structure to transfer RSA+AES encrypted data via HTTP request body.
// - Key contains AES algorithm key encrypted with RSA public key and BASE64 encoded.
// - Cipher contains original []byte payload encrypted with the AES key and BASE64 encoded.
// - Nonce contains random nonce string used for AES encryption, BASE64 encoded.
type rsaAESEncryptedBlock struct {
	Key    string `json:"key"`
	Cipher string `json:"cipher"`
	Nonce  string `json:"nonce"`
}

func (engine *RSAAESEngine) Encrypt(data []byte) ([]byte, error) {
	if engine.publicKey == nil {
		return nil, ErrEncryptingDisabled
	}

	aesEncData, err := aesEncrypt(data)
	if err != nil {
		return nil, fmt.Errorf("AES encrypting error: %w", err)
	}

	rsaEncKey, err := rsa.EncryptOAEP(
		sha256.New(), rand.Reader, engine.publicKey, aesEncData.Key, engine.label)
	if err != nil {
		return nil, fmt.Errorf("RSA encrypting error: %w", err)
	}

	rsaAESEncData := &rsaAESEncryptedBlock{
		Key:    base64.StdEncoding.EncodeToString(rsaEncKey),
		Cipher: base64.StdEncoding.EncodeToString(aesEncData.Ciphertext),
		Nonce:  base64.StdEncoding.EncodeToString(aesEncData.Nonce),
	}

	encryptedBytes, err := json.Marshal(rsaAESEncData)
	if err != nil {
		return nil, fmt.Errorf("JSON marshalling error: %w", err)
	}
	return encryptedBytes, nil
}

func (engine *RSAAESEngine) EncryptingEnabled() bool {
	return engine.publicKey != nil
}

func (engine *RSAAESEngine) Decrypt(encrypted []byte) ([]byte, error) {
	if engine.privateKey == nil {
		return nil, ErrDecryptingDisabled
	}

	rsaAESEncData := &rsaAESEncryptedBlock{}
	err := json.Unmarshal(encrypted, rsaAESEncData)
	if err != nil {
		return nil, fmt.Errorf("JSON unmarshalling error: %w", err)
	}

	rsaEncKey, err := base64.StdEncoding.DecodeString(rsaAESEncData.Key)
	if err != nil {
		return nil, fmt.Errorf("BASE64 decoding error: %w", err)
	}

	aesEncData := &aesEncrypted{}
	aesEncData.Key, err = rsa.DecryptOAEP(sha256.New(), nil, engine.privateKey, rsaEncKey, engine.label)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt error: %w", err)
	}

	aesEncData.Ciphertext, err = base64.StdEncoding.DecodeString(rsaAESEncData.Cipher)
	if err != nil {
		return nil, fmt.Errorf("BASE64 decoding error: %w", err)
	}

	aesEncData.Nonce, err = base64.StdEncoding.DecodeString(rsaAESEncData.Nonce)
	if err != nil {
		return nil, fmt.Errorf("BASE64 decoding error: %w", err)
	}

	data, err := aesDecrypt(aesEncData)
	if err != nil {
		return nil, fmt.Errorf("AES decrypt error: %w", err)
	}
	return data, nil
}

func (engine *RSAAESEngine) DecryptingEnabled() bool {
	return engine.privateKey != nil
}

// RSAEngineBuilder is a builder structure for RSAAESEngine which can provide any combination of private and public keys
// depending on functional needed from the engine
type RSAEngineBuilder struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
	label      []byte
}

func NewRSAEngineBuilder() *RSAEngineBuilder {
	return &RSAEngineBuilder{}
}

func (b *RSAEngineBuilder) PublicKey(pub *rsa.PublicKey) *RSAEngineBuilder {
	b.publicKey = pub
	return b
}

func (b *RSAEngineBuilder) PrivateKey(priv *rsa.PrivateKey) *RSAEngineBuilder {
	b.privateKey = priv
	return b
}

func (b *RSAEngineBuilder) Label(label []byte) *RSAEngineBuilder {
	b.label = label
	return b
}

func (b *RSAEngineBuilder) Build() *RSAAESEngine {
	label := b.label
	if label == nil {
		label = []byte("")
	}
	return &RSAAESEngine{
		publicKey:  b.publicKey,
		privateKey: b.privateKey,
		label:      label,
	}
}

// ReadRSAPublicKeyFromFile is utility function to read an RSA public key from file
func ReadRSAPublicKeyFromFile(filePath string) (*rsa.PublicKey, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading key file %s: %w", filePath, err)
	}

	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != publicKeyBlockType {
		return nil, fmt.Errorf("error reading key file %s: invalid PEM block or key type", filePath)
	}

	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing public key: %w", err)
	}

	return pub, nil
}

// ReadRSAPrivateKeyFromFile is utility function to read an RSA private key from file
func ReadRSAPrivateKeyFromFile(filePath string) (*rsa.PrivateKey, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading key file %s: %w", filePath, err)
	}

	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != privateKeyBlockType {
		return nil, fmt.Errorf("error reading key file %s: invalid PEM block or key type", filePath)
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing public key: %w", err)
	}

	return priv, nil
}
