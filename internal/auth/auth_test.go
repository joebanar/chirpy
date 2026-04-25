package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashAndCheck(t *testing.T) {
	p := "super-secret"
	h, err := HashPassword(p)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if h == p {
		t.Fatalf("hashed password should not equal raw password")
	}
	ok, err := CheckPasswordHash(p, h)
	if err != nil {
		t.Fatalf("CheckPasswordHash error: %v", err)
	}
	if !ok {
		t.Fatalf("password did not match hash")
	}
}

func TestMakeAndValidateJWT(t *testing.T) {
	id := uuid.New()
	secret := "test-secret"
	tok, err := MakeJWT(id, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}
	got, err := ValidateJWT(tok, secret)
	if err != nil {
		t.Fatalf("ValidateJWT error: %v", err)
	}
	if got != id {
		t.Fatalf("expected id %s, got %s", id, got)
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	id := uuid.New()
	secret := "expire-secret"
	tok, err := MakeJWT(id, secret, -time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}
	if _, err := ValidateJWT(tok, secret); err == nil {
		t.Fatalf("expected error validating expired token, got nil")
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	id := uuid.New()
	secret := "right-secret"
	tok, err := MakeJWT(id, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}
	if _, err := ValidateJWT(tok, "wrong-secret"); err == nil {
		t.Fatalf("expected error validating token with wrong secret, got nil")
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"Bearer abc.def.ghi"}
	h := make(http.Header)
	for k, v := range headers {
		for _, vv := range v {
			h.Add(k, vv)
		}
	}
	tok, err := GetBearerToken(h)
	if err != nil {
		t.Fatalf("GetBearerToken error: %v", err)
	}
	if tok != "abc.def.ghi" {
		t.Fatalf("expected token 'abc.def.ghi', got %q", tok)
	}
}
