package api

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	password := "correct horse battery staple"
	encoded := hashPassword(password)
	if encoded == password {
		t.Fatal("password was not hashed")
	}
	if !verifyPassword(password, encoded) {
		t.Fatal("expected password hash to verify")
	}
	if verifyPassword("incorrect password", encoded) {
		t.Fatal("unexpected password verification success")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	if verifyPassword("anything", "not-a-valid-hash") {
		t.Fatal("malformed hash must not verify")
	}
}
