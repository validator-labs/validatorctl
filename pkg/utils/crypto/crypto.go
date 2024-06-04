package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"

	prompt_utils "github.com/spectrocloud-labs/prompts-tui/prompts"

	log "github.com/validator-labs/validatorctl/pkg/logging"
)

// TODO: should this value be changed? Is it safe to have it in the code and not in some env var that can be configured on build?
const k = "TxXW4Qg4vqorgiCtgeEFW7inLXKLv4bC"

func DecryptB64(cipherstringB64 string) (*[]byte, error) {
	if cipherstringB64 == "" {
		return new([]byte), nil
	}

	encrypted, err := base64.StdEncoding.DecodeString(cipherstringB64)
	if err != nil {
		return nil, err
	}

	ciphertext := []byte(encrypted)
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

func ReadCACert(prompt string, defaultCaCertPath, caCertPathOverride string) (caCertPath string, caCertName string, caCertData []byte, err error) {
	if caCertPathOverride != "" {
		caCertPath = caCertPathOverride
	} else {
		log.InfoCLI("Optionally enter the file path to your desired CA certificate, e.g., /usr/local/share/ca-certificates/ca.crt")
		log.InfoCLI("Press enter to skip if your certificates are publicly verifiable")
		caCertPath, err = prompt_utils.ReadFilePath(prompt, defaultCaCertPath, "Invalid filepath specified", true)
	}
	if err != nil {
		return "", "", nil, err
	}
	if caCertPath == "" {
		return "", "", nil, nil
	}
	caFile, _ := os.Stat(caCertPath)
	caBytes, err := os.ReadFile(caCertPath) //#nosec
	if err != nil {
		return "", "", nil, err
	}
	// Validate CA cert
	var blocks []byte
	rest := caBytes
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return "", "", nil, fmt.Errorf("PEM parse failure for %s", caCertPath)
		}
		blocks = append(blocks, block.Bytes...)
		if len(rest) == 0 {
			break
		}
	}
	if _, err = x509.ParseCertificates(blocks); err != nil {
		return "", "", nil, err
	}
	return caCertPath, caFile.Name(), caBytes, nil
}
