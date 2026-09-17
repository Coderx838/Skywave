package crypto

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	passphrase := "super-secret-room-pass-42"
	original := "Hello from an encrypted anonymous terminal room! 🌊"

	cipherText, err := Encrypt(original, passphrase)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if cipherText == original {
		t.Fatalf("Ciphertext cannot match original plaintext")
	}

	decrypted, err := Decrypt(cipherText, passphrase)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != original {
		t.Fatalf("Expected decrypted %q, got %q", original, decrypted)
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	cipherText, _ := Encrypt("Confidential data", "correct-password")
	_, err := Decrypt(cipherText, "wrong-password")
	if err == nil {
		t.Fatalf("Expected decryption error with wrong password, but got nil")
	}
}

func TestHashSecret(t *testing.T) {
	h1 := HashSecret("mypassword", "#secret")
	h2 := HashSecret("mypassword", "#secret")
	h3 := HashSecret("otherpassword", "#secret")

	if h1 != h2 {
		t.Fatalf("Same inputs should yield matching hashes")
	}
	if h1 == h3 {
		t.Fatalf("Different passwords should yield different hashes")
	}
}
