package api

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type loginAttemptWindow struct {
	started time.Time
	count   int
}

type loginRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]loginAttemptWindow
	now     func() time.Time
}

func newLoginRateLimiter(limit int, window time.Duration) *loginRateLimiter {
	return &loginRateLimiter{
		limit: limit, window: window,
		clients: make(map[string]loginAttemptWindow),
		now: time.Now,
	}
}

func (l *loginRateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r.RemoteAddr)
		allowed, retryAfter := l.allow(key)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *loginRateLimiter) allow(key string) (bool, int) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.clients[key]
	if !exists || now.Sub(entry.started) >= l.window {
		if len(l.clients) >= 10000 {
			for client, candidate := range l.clients {
				if now.Sub(candidate.started) >= l.window {
					delete(l.clients, client)
				}
			}
		}
		if len(l.clients) >= 10000 {
			return false, int(l.window.Seconds())
		}
		l.clients[key] = loginAttemptWindow{started: now, count: 1}
		return true, 0
	}
	if entry.count >= l.limit {
		remaining := l.window - now.Sub(entry.started)
		retryAfter := int(remaining.Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}
	entry.count++
	l.clients[key] = entry
	return true, 0
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}
