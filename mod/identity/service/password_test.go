package service

import (
	"testing"
)

func TestPasswordHashAndVerify(t *testing.T) {
	p := NewPassword(10)

	hash, err := p.Hash("correct horse battery")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "correct horse battery" {
		t.Fatal("hash must not equal the plain password")
	}
	if err := p.Verify(hash, "correct horse battery"); err != nil {
		t.Errorf("verify of the right password failed: %v", err)
	}
	if err := p.Verify(hash, "wrong password"); err == nil {
		t.Error("verify of a wrong password must fail")
	}
}

func TestPasswordHashIsSalted(t *testing.T) {
	p := NewPassword(10)

	a, err := p.Hash("same password")
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Hash("same password")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("two hashes of the same password must differ (unique salt)")
	}
}

func TestPasswordPolicy(t *testing.T) {
	p := NewPassword(10)

	if err := p.Validate("short"); err == nil {
		t.Error("short password must be rejected")
	}
	if err := p.Validate("1234567890"); err != nil {
		t.Errorf("10 characters must be accepted: %v", err)
	}

	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	if err := p.Validate(string(long)); err == nil {
		t.Error("a password longer than 72 bytes must be rejected")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	p := NewPassword(10)

	for _, bad := range []string{"", "plain", "$argon2id$broken", "$$$$$", "$bcrypt$v=1$m=1,t=1,p=1$a$b"} {
		if err := p.Verify(bad, "anything"); err == nil {
			t.Errorf("malformed hash %q must not verify", bad)
		}
	}
}
