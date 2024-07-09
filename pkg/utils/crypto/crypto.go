// Package crypto contains utility functions for encryption and decryption.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const k = "dJsv0hrLaXowK81xA2bEjfoubfkmDwIL"

// DecryptB64 decrypts a base64-encoded string.
func DecryptB64(cipherstringB64 string) (*[]byte, error) {
	if cipherstringB64 == "" {
		return new([]byte), nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(cipherstringB64)
	if err != nil {
		return nil, err
	}

	key := []byte(k)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("cipherstring is too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return &ciphertext, nil
}

// EncryptB64 encrypts a byte slice and returns a base64-encoded string.
func EncryptB64(plainbytes []byte) (string, error) {
	if len(plainbytes) == 0 {
		return "", nil
	}

	key := []byte(k)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, aes.BlockSize+len(plainbytes))

	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plainbytes)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
