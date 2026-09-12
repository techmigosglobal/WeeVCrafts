package auth

import "testing"

func TestArgon2idHasherRoundTrip(t *testing.T) {
	hasher := NewArgon2idHasher()
	encoded, err := hasher.Hash("a-long-enough-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !hasher.Compare(encoded, "a-long-enough-password") {
		t.Fatal("expected password to verify")
	}
	if hasher.Compare(encoded, "wrong-password") {
		t.Fatal("wrong password verified")
	}
	if hasher.Compare("not-an-argon2id-hash", "a-long-enough-password") {
		t.Fatal("malformed hash verified")
	}
}
