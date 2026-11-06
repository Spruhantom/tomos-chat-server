package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginRateLimiterBlocksAfterLimit(t *testing.T) {
	limiter := newLoginRateLimiter(2, time.Minute)
	called := 0
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		request.RemoteAddr = "192.0.2.10:12345"
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d: expected 204, got %d", i+1, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	request.RemoteAddr = "192.0.2.10:54321"
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if called != 2 {
		t.Fatalf("handler called %d times; want 2", called)
	}
}

func TestLoginRateLimiterSeparatesClientIPsAndIgnoresForwardedHeader(t *testing.T) {
	limiter := newLoginRateLimiter(1, time.Minute)
	called := 0
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))

	requests := []struct {
		remote string
		forwarded string
		wantStatus int
	}{
		{remote: "192.0.2.1:1000", forwarded: "198.51.100.99", wantStatus: http.StatusNoContent},
		{remote: "192.0.2.1:1001", forwarded: "203.0.113.50", wantStatus: http.StatusTooManyRequests},
		{remote: "192.0.2.2:1000", forwarded: "198.51.100.99", wantStatus: http.StatusNoContent},
	}
	for _, item := range requests {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		request.RemoteAddr = item.remote
		request.Header.Set("X-Forwarded-For", item.forwarded)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != item.wantStatus {
			t.Fatalf("remote %s: expected %d, got %d", item.remote, item.wantStatus, recorder.Code)
		}
	}
	if called != 2 {
		t.Fatalf("handler called %d times; want 2", called)
	}
}

func TestLoginRateLimiterResetsExpiredWindow(t *testing.T) {
	limiter := newLoginRateLimiter(1, time.Minute)
	now := time.Now()
	limiter.now = func() time.Time { return now }
	if allowed, _ := limiter.allow("192.0.2.3"); !allowed {
		t.Fatal("first attempt should be allowed")
	}
	if allowed, _ := limiter.allow("192.0.2.3"); allowed {
		t.Fatal("second attempt should be blocked")
	}
	now = now.Add(time.Minute)
	if allowed, _ := limiter.allow("192.0.2.3"); !allowed {
		t.Fatal("attempt after window should be allowed")
	}
}
