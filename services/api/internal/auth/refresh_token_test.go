package auth

import (
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestNewRefreshTokenUnique(t *testing.T) {
	allure.Wrap(t, func(ctx *allure.Context) {
		tokenA, hashA, err := NewRefreshToken()
		if err != nil {
			t.Fatalf("NewRefreshToken: %v", err)
		}
		tokenB, hashB, err := NewRefreshToken()
		if err != nil {
			t.Fatalf("NewRefreshToken: %v", err)
		}
		if tokenA == tokenB {
			t.Fatal("expected unique refresh tokens")
		}
		if hashA == hashB {
			t.Fatal("expected unique refresh token hashes")
		}
		if HashRefreshToken(tokenA) != hashA {
			t.Fatal("HashRefreshToken mismatch")
		}
		_ = ctx
	})
}

func TestNewMobileAuthCodeUnique(t *testing.T) {
	allure.Wrap(t, func(ctx *allure.Context) {
		codeA, hashA, err := NewMobileAuthCode()
		if err != nil {
			t.Fatalf("NewMobileAuthCode: %v", err)
		}
		codeB, hashB, err := NewMobileAuthCode()
		if err != nil {
			t.Fatalf("NewMobileAuthCode: %v", err)
		}
		if codeA == codeB {
			t.Fatal("expected unique auth codes")
		}
		if hashA == hashB {
			t.Fatal("expected unique auth code hashes")
		}
		if HashMobileAuthCode(codeA) != hashA {
			t.Fatal("HashMobileAuthCode mismatch")
		}
		_ = ctx
	})
}

func TestDefaultTTLs(t *testing.T) {
	allure.Wrap(t, func(ctx *allure.Context) {
		if DefaultRefreshTokenTTL < 24*time.Hour {
			t.Fatalf("refresh token TTL too short: %v", DefaultRefreshTokenTTL)
		}
		if DefaultMobileAuthCodeTTL > 15*time.Minute {
			t.Fatalf("auth code TTL too long: %v", DefaultMobileAuthCodeTTL)
		}
		_ = ctx
	})
}
