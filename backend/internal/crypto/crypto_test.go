package crypto

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("ENCRYPTION_KEY", "test-encryption-key-for-testing!!")
	InitDefault()
	os.Exit(m.Run())
}

func TestEncryptor_RoundTrip(t *testing.T) {
	enc := NewEncryptorWithKey([]byte("test-key-for-round-trip-testing!"))

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"simple", "hello"},
		{"special_chars", "!@#$%^&*()"},
		{"unicode", "한글테스트"},
		{"long_text", "this is a very long text that should still work correctly with encryption"},
		{"api_token", "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := enc.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			if encrypted == tc.plaintext {
				t.Error("Encrypted text should not equal plaintext")
			}

			decrypted, err := enc.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Decrypted text mismatch: expected '%s', got '%s'", tc.plaintext, decrypted)
			}
		})
	}
}

func TestEncryptor_Uniqueness(t *testing.T) {
	enc := NewEncryptorWithKey([]byte("test-key-for-uniqueness-testing!"))
	plaintext := "same_text"

	encrypted1, _ := enc.Encrypt(plaintext)
	encrypted2, _ := enc.Encrypt(plaintext)

	if encrypted1 == encrypted2 {
		t.Error("Encrypting same text twice should produce different ciphertexts")
	}

	decrypted1, _ := enc.Decrypt(encrypted1)
	decrypted2, _ := enc.Decrypt(encrypted2)

	if decrypted1 != decrypted2 {
		t.Error("Both ciphertexts should decrypt to the same plaintext")
	}
}

func TestEncryptor_DecryptErrors(t *testing.T) {
	enc := NewEncryptorWithKey([]byte("test-key-for-decrypt-error-test!"))

	testCases := []struct {
		name       string
		ciphertext string
		wantErr    error
	}{
		{"empty", "", ErrEmptyCiphertext},
		{"invalid_base64", "not-valid-base64!!!", nil}, // base64 decode error
		{"tampered", "dGFtcGVyZWRkYXRhdGhhdGlzbm90dmFsaWQ=", nil}, // will fail decrypt
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := enc.Decrypt(tc.ciphertext)
			if err == nil {
				t.Error("Expected error for invalid ciphertext")
			}
			if tc.wantErr != nil && err != tc.wantErr {
				t.Errorf("Expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestEncryptorWithCustomKey(t *testing.T) {
	key := []byte("my-custom-key-for-testing-12345")
	enc := NewEncryptorWithKey(key)

	plaintext := "test message"
	encrypted, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestPackageFunctions(t *testing.T) {
	plaintext := "test with package functions"

	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected '%s', got '%s'", plaintext, decrypted)
	}
}
