package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

const (
	keyEnvVar = "ENCRYPTION_KEY"
	keySize   = 32 // AES-256
)

var (
	ErrEmptyCiphertext = errors.New("empty ciphertext")
	ErrCiphertextShort = errors.New("ciphertext too short")
	ErrDecryptFailed   = errors.New("decryption failed")
	ErrKeyNotSet       = errors.New("ENCRYPTION_KEY environment variable is not set")
)

type Encryptor struct {
	key []byte
}

func NewEncryptor() (*Encryptor, error) {
	key, err := getKey()
	if err != nil {
		return nil, err
	}
	return &Encryptor{key: key}, nil
}

func NewEncryptorWithKey(key []byte) *Encryptor {
	if len(key) < keySize {
		padded := make([]byte, keySize)
		copy(padded, key)
		key = padded
	} else if len(key) > keySize {
		key = key[:keySize]
	}
	return &Encryptor{key: key}
}

func getKey() ([]byte, error) {
	key := os.Getenv(keyEnvVar)
	if key == "" {
		return nil, ErrKeyNotSet
	}

	keyBytes := []byte(key)
	if len(keyBytes) < keySize {
		padded := make([]byte, keySize)
		copy(padded, keyBytes)
		return padded, nil
	}
	return keyBytes[:keySize], nil
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", ErrEmptyCiphertext
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", ErrCiphertextShort
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptFailed
	}

	return string(plaintext), nil
}

func InitDefault() error {
	enc, err := NewEncryptor()
	if err != nil {
		return err
	}
	defaultEncryptor = enc
	return nil
}

var defaultEncryptor *Encryptor

func Encrypt(plaintext string) (string, error) {
	if defaultEncryptor == nil {
		return "", ErrKeyNotSet
	}
	return defaultEncryptor.Encrypt(plaintext)
}

func Decrypt(ciphertext string) (string, error) {
	if defaultEncryptor == nil {
		return "", ErrKeyNotSet
	}
	return defaultEncryptor.Decrypt(ciphertext)
}
