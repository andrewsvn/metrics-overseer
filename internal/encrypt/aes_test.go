package encrypt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAESEncryption(t *testing.T) {
	srctext, err := generateRandomBytes(2000)
	require.NoError(t, err)

	encrypted, err := aesEncrypt(srctext)
	require.NoError(t, err)

	decrypted, err := aesDecrypt(encrypted)
	require.NoError(t, err)
	require.Equal(t, srctext, decrypted)
}
