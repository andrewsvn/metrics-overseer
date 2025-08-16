package encrypt

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
)

const (
	SignHeader = "HashSHA256"
)

var (
	ErrSignatureInvalid = errors.New("request signature invalid")
)

func AddSignature(key []byte, payload []byte, header http.Header) {
	if len(key) == 0 {
		return
	}

	sign := calculateHash(key, payload)
	header.Add(SignHeader, base64.StdEncoding.EncodeToString(sign))
}

func GetSignature(header http.Header) ([]byte, error) {
	signstr := header.Get(SignHeader)
	if len(signstr) == 0 {
		return nil, nil
	}

	sign, err := base64.StdEncoding.DecodeString(signstr)
	if err != nil {
		return nil, fmt.Errorf("unable to decode signature from header: %w", err)
	}
	return sign, nil
}

func CheckSignature(key []byte, payload []byte, sign []byte) error {
	calculated := calculateHash(key, payload)
	if !bytes.Equal(calculated, sign) {
		return ErrSignatureInvalid
	}
	return nil
}

func calculateHash(key []byte, payload []byte) []byte {
	hfunc := sha256.New()
	if payload != nil {
		hfunc.Write(payload)
	}
	return hfunc.Sum(key)
}
