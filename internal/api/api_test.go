package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	password := "correct horse battery staple"
	encoded, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
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


func TestDecodeJSONRejectsUnknownFieldsAndTrailingValues(t *testing.T) {
	for _, body := range []string{
		`{"username":"tom","password":"long-password-value","extra":true}`,
		`{"username":"tom","password":"long-password-value"} {"another":"object"}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var input credentials
		if decodeJSON(recorder, request, &input) {
			t.Fatalf("expected invalid JSON to be rejected: %s", body)
		}
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	}
}

func TestSecurityHeadersAreApplied(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy": "no-referrer",
		"Cache-Control": "no-store",
	} {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
