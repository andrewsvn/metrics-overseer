package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSAKeyPairGeneration(t *testing.T) {
	const (
		baseDir  = "key_testdata"
		privName = "priv.pem"
		pubName  = "pub.pem"
	)

	err := os.MkdirAll(baseDir, 0755)
	require.NoError(t, err)

	// RSA key size can't be less than 1024 bits
	err = generateKeyPair(baseDir, privName, pubName, 512)
	assert.Error(t, err)

	err = generateKeyPair(baseDir, privName, pubName, 1024)
	assert.NoError(t, err)

	checkKeyFile(t, filepath.Join(baseDir, privName), "RSA PRIVATE KEY")
	checkKeyFile(t, filepath.Join(baseDir, pubName), "PUBLIC KEY")

	_ = os.RemoveAll(baseDir)
}

func checkKeyFile(t *testing.T, filePath string, keyType string) {
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	require.NoError(t, err)
	defer func(f *os.File) {
		err := f.Close()
		assert.NoError(t, err)
	}(f)

	scanner := bufio.NewScanner(f)
	beginKey := "BEGIN " + keyType
	endKey := "END " + keyType
	hasBeginKey := false
	hasEndKey := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), beginKey) {
			hasBeginKey = true
			continue
		}
		if strings.Contains(scanner.Text(), endKey) {
			hasEndKey = true
		}
	}
	assert.True(t, hasBeginKey)
	assert.True(t, hasEndKey)
}
