package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func generateKeyPair(baseDir string, privateKeyName string, publicKeyName string, nBits int) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, nBits)
	if err != nil {
		return fmt.Errorf("error generating RSA key pair: %w", err)
	}

	var privateKeyPem bytes.Buffer
	err = pem.Encode(&privateKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return fmt.Errorf("error encoding RSA private key: %w", err)
	}

	var publicKeyPem bytes.Buffer
	err = pem.Encode(&publicKeyPem, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})
	if err != nil {
		return fmt.Errorf("error encoding RSA public key: %w", err)
	}

	err = os.WriteFile(filepath.Join(baseDir, privateKeyName), privateKeyPem.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing RSA private key: %w", err)
	}

	err = os.WriteFile(filepath.Join(baseDir, publicKeyName), publicKeyPem.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error writing RSA public key: %w", err)
	}

	return nil
}

func main() {
	var baseDir string
	var privKeyName string
	var pubKeyName string
	var bits int

	flag.StringVar(&privKeyName, "pr", "private.pem", "File name for private key")
	flag.StringVar(&pubKeyName, "pb", "public.pem", "File name for public key")
	flag.IntVar(&bits, "bits", 2048, "Key pair bit size (minimum 1024, not less than 2048 recommended)")
	flag.Parse()

	baseDir = flag.Arg(0)
	if baseDir == "" {
		baseDir = "."
	}

	err := generateKeyPair(baseDir, privKeyName, pubKeyName, bits)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("keys successfully generated in %s\n", baseDir)
}
