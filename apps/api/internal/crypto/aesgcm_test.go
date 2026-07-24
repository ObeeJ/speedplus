package crypto

import (
	"strings"
	"testing"
)

func testCipher(t *testing.T) *Cipher {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c := testCipher(t)
	plaintext := "Ada Okafor +2348012345678"

	enc, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == plaintext {
		t.Fatal("ciphertext must not equal plaintext")
	}
	if strings.Contains(enc, "Ada") {
		t.Fatal("ciphertext leaks plaintext content")
	}

	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if dec != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", dec, plaintext)
	}
}

func TestEncryptIsNonDeterministic(t *testing.T) {
	c := testCipher(t)
	a, _ := c.Encrypt("same input")
	b, _ := c.Encrypt("same input")
	if a == b {
		t.Fatal("two encryptions of the same plaintext must differ (random nonce)")
	}
}

func TestEmptyStringRoundTrip(t *testing.T) {
	c := testCipher(t)
	enc, err := c.Encrypt("")
	if err != nil || enc != "" {
		t.Fatalf("empty plaintext should encrypt to empty string, got %q err=%v", enc, err)
	}
	dec, err := c.Decrypt("")
	if err != nil || dec != "" {
		t.Fatalf("empty ciphertext should decrypt to empty string, got %q err=%v", dec, err)
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	c := testCipher(t)
	enc, _ := c.Encrypt("sensitive phone number")
	tampered := enc[:len(enc)-4] + "AAAA"
	if _, err := c.Decrypt(tampered); err == nil {
		t.Fatal("decrypting tampered ciphertext must fail (GCM auth tag)")
	}
}

func TestWrongKeyCannotDecrypt(t *testing.T) {
	c1 := testCipher(t)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(255 - i)
	}
	c2, err := NewCipher(key2)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	enc, _ := c1.Encrypt("secret")
	if _, err := c2.Decrypt(enc); err == nil {
		t.Fatal("decrypting with the wrong key must fail")
	}
}

func TestInvalidKeySize(t *testing.T) {
	if _, err := NewCipher([]byte("too short")); err != ErrInvalidKeySize {
		t.Fatalf("expected ErrInvalidKeySize, got %v", err)
	}
}

func TestEncryptPtrDecryptPtr(t *testing.T) {
	c := testCipher(t)

	if p, err := c.EncryptPtr(nil); err != nil || p != nil {
		t.Fatalf("EncryptPtr(nil) should return nil, nil; got %v, %v", p, err)
	}
	if p, err := c.DecryptPtr(nil); err != nil || p != nil {
		t.Fatalf("DecryptPtr(nil) should return nil, nil; got %v, %v", p, err)
	}

	name := "Emeka Nwosu"
	enc, err := c.EncryptPtr(&name)
	if err != nil {
		t.Fatalf("EncryptPtr: %v", err)
	}
	if enc == nil || *enc == name {
		t.Fatal("EncryptPtr must return a distinct ciphertext pointer")
	}
	dec, err := c.DecryptPtr(enc)
	if err != nil {
		t.Fatalf("DecryptPtr: %v", err)
	}
	if dec == nil || *dec != name {
		t.Fatalf("round trip mismatch: got %v want %q", dec, name)
	}
}
