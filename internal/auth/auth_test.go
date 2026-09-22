package auth

import (
	"testing"
	"time"
)

func TestSignAndVerifySession(t *testing.T) {
	a := &Auth{cfg: Config{SessionKey: []byte("test-secret")}}

	want := time.Now().Add(time.Hour).Truncate(time.Second)
	value, err := a.signSession(want)
	if err != nil {
		t.Fatalf("signSession: %v", err)
	}

	got, ok := a.verifySession(value)
	if !ok || !got.Equal(want) {
		t.Fatalf("verifySession() = %v, %v; want %v, true", got, ok, want)
	}
}

func TestVerifySessionRejectsTampering(t *testing.T) {
	a := &Auth{cfg: Config{SessionKey: []byte("test-secret")}}

	value, err := a.signSession(time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("signSession: %v", err)
	}

	other := &Auth{cfg: Config{SessionKey: []byte("different-secret")}}
	if _, ok := other.verifySession(value); ok {
		t.Fatal("verifySession() accepted a cookie signed with a different key")
	}
	if _, ok := a.verifySession(value + "tampered"); ok {
		t.Fatal("verifySession() accepted a tampered cookie")
	}
	if _, ok := a.verifySession("garbage"); ok {
		t.Fatal("verifySession() accepted malformed input")
	}
}
