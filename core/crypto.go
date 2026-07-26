package core

import (
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// octoEncHeader is the magic string indicating an encrypted state file.
	octoEncHeader = "OCTOENC1"
	saltSize      = 16
)

// Encrypt payload using Argon2id and ChaCha20-Poly1305.
// The output format is: [Header (8)] [Salt (16)] [Nonce (24)] [Ciphertext]
func Encrypt(data []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, errors.New("crypto: empty password")
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := deriveKey(password, salt)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, data, nil)

	// Construct payload
	out := make([]byte, 0, len(octoEncHeader)+saltSize+len(nonce)+len(ciphertext))
	out = append(out, []byte(octoEncHeader)...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	return out, nil
}

// Decrypt attempts to decrypt an OCTOENC1 payload.
func Decrypt(payload []byte, password string) ([]byte, error) {
	headerLen := len(octoEncHeader)
	nonceSize := chacha20poly1305.NonceSizeX
	minLen := headerLen + saltSize + nonceSize + 16 // 16 is Poly1305 tag size

	if len(payload) < minLen {
		return nil, errors.New("crypto: invalid or corrupted encrypted payload")
	}

	if string(payload[:headerLen]) != octoEncHeader {
		return nil, errors.New("crypto: not an encrypted payload")
	}

	salt := payload[headerLen : headerLen+saltSize]
	nonceStart := headerLen + saltSize
	nonce := payload[nonceStart : nonceStart+nonceSize]
	ciphertext := payload[nonceStart+nonceSize:]

	key := deriveKey(password, salt)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("crypto: incorrect password or corrupted data")
	}

	return plaintext, nil
}

// IsEncrypted fast-checks if a byte slice starts with the encryption header.
func IsEncrypted(payload []byte) bool {
	return len(payload) >= len(octoEncHeader) && string(payload[:len(octoEncHeader)]) == octoEncHeader
}

// deriveKey uses Argon2id to derive a 32-byte key from the password and salt.
// Parameters: time=1, memory=64MB, threads=4, keyLen=32
func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
}
