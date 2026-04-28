package crypto

import "testing"

func TestEncryptDecryptRoundtrip(t *testing.T) {
	encryptor, err := NewEncryptor("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("NewEncryptor returned error: %v", err)
	}

	ciphertext, err := encryptor.Encrypt("super-secret")
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}

	if ciphertext == "super-secret" {
		t.Fatal("expected ciphertext to differ from plaintext")
	}

	plaintext, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}

	if plaintext != "super-secret" {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}
}

func TestDecryptLegacyPlaintext(t *testing.T) {
	encryptor, err := NewEncryptor("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("NewEncryptor returned error: %v", err)
	}

	plaintext, err := encryptor.Decrypt("legacy-password")
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}

	if plaintext != "legacy-password" {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}
}
