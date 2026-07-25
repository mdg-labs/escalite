package auth

import (
	"testing"
	"time"
)

func TestNewRefreshTokenUnique(t *testing.T) {
	a, hashA, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	b, hashB, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if a == b {
		t.Fatal("expected unique refresh tokens")
	}
	if hashA == hashB {
		t.Fatal("expected unique refresh token hashes")
	}
	if HashRefreshToken(a) != hashA {
		t.Fatal("HashRefreshToken mismatch")
	}
}

func TestNewMobileAuthCodeUnique(t *testing.T) {
	a, hashA, err := NewMobileAuthCode()
	if err != nil {
		t.Fatalf("NewMobileAuthCode: %v", err)
	}
	b, hashB, err := NewMobileAuthCode()
	if err != nil {
		t.Fatalf("NewMobileAuthCode: %v", err)
	}
	if a == b {
		t.Fatal("expected unique auth codes")
	}
	if hashA == hashB {
		t.Fatal("expected unique auth code hashes")
	}
	if HashMobileAuthCode(a) != hashA {
		t.Fatal("HashMobileAuthCode mismatch")
	}
}

func TestDefaultTTLs(t *testing.T) {
	if DefaultRefreshTokenTTL < 24*time.Hour {
		t.Fatalf("refresh token TTL too short: %v", DefaultRefreshTokenTTL)
	}
	if DefaultMobileAuthCodeTTL > 15*time.Minute {
		t.Fatalf("auth code TTL too long: %v", DefaultMobileAuthCodeTTL)
	}
}
