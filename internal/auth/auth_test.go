package auth

import "testing"

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
