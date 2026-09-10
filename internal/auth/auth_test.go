package auth

import (
	"net/http/httptest"
	"testing"
)

const testHash = "$argon2id$v=19$m=19456,t=2,p=1$bIQ5JeCLKlP/HKx9tLSxLQ$7myOb/wrmOBE639j16R+0MdwwjUsG2oNNtQem3OYtzs"

func TestSignedSession(t *testing.T) {
	a, err := New("owner", testHash, "integration-test-secret-at-least-32-bytes", false)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Verify("owner", "testtesttest12") || a.Verify("owner", "wrong") {
		t.Fatal("credential verification mismatch")
	}
	response := httptest.NewRecorder()
	if err := a.Login(response); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/admin", nil)
	request.AddCookie(response.Result().Cookies()[0])
	session, ok := a.Session(request)
	if !ok || session.CSRF == "" {
		t.Fatal("signed session was rejected")
	}
	cookie := response.Result().Cookies()[0]
	cookie.Value += "tampered"
	forged := httptest.NewRequest("GET", "/admin", nil)
	forged.AddCookie(cookie)
	if _, ok := a.Session(forged); ok {
		t.Fatal("tampered session was accepted")
	}
}
