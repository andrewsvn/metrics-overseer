package encrypt

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSAEngineUnlabeled(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	encrypter := NewRSAEngineBuilder().PublicKey(&priv.PublicKey).Build()
	decrypter := NewRSAEngineBuilder().PrivateKey(priv).Build()

	// sunny day scenario
	data := []byte("a quick brown fox jumps over the lazy dog")
	encrypted, err := encrypter.Encrypt(data)
	assert.NoError(t, err)
	decrypted, err := decrypter.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, data, decrypted)

	priv, err = rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// decryption/encryption enabled
	assert.True(t, encrypter.EncryptingEnabled())
	assert.False(t, encrypter.DecryptingEnabled())
	assert.False(t, decrypter.EncryptingEnabled())
	assert.True(t, decrypter.DecryptingEnabled())
	encrypted, err = decrypter.Encrypt(data)
	assert.ErrorAs(t, err, &ErrEncryptingDisabled)
	_, err = encrypter.Decrypt(encrypted)
	assert.ErrorAs(t, err, &ErrDecryptingDisabled)

	// big data chunk
	data = make([]byte, 4000)
	_, _ = rand.Read(data)
	encrypted, err = encrypter.Encrypt(data)
	assert.NoError(t, err)
	decrypted, err = decrypter.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, data, decrypted)

	// rainy day scenario - different keys
	encrypter = NewRSAEngineBuilder().PublicKey(&priv.PublicKey).Build()

	encrypted, err = encrypter.Encrypt(data)
	assert.NoError(t, err)
	_, err = decrypter.Decrypt(encrypted)
	assert.Error(t, err)
}

func TestRSAEngineLabeled(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	label := []byte("mylabel")

	// sunny day scenario
	encrypter := NewRSAEngineBuilder().PublicKey(&priv.PublicKey).Label(label).Build()
	decrypter := NewRSAEngineBuilder().PrivateKey(priv).Label(label).Build()
	data := []byte("a quick brown fox jumps over the lazy dog")
	encrypted, err := encrypter.Encrypt(data)
	assert.NoError(t, err)
	decrypted, err := decrypter.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, data, decrypted)

	// rainy day scenario - different labels
	decrypter = NewRSAEngineBuilder().PrivateKey(priv).Build()
	encrypted, err = encrypter.Encrypt(data)
	assert.NoError(t, err)
	_, err = decrypter.Decrypt(encrypted)
	assert.Error(t, err)
}
