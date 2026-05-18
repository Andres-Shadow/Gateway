package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientBucket
	limit    int
	interval time.Duration
}

type clientBucket struct {
	count     int
	resetTime time.Time
}

func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 120
	}
	if interval <= 0 {
		interval = time.Minute
	}
	return &RateLimiter{clients: make(map[string]*clientBucket), limit: limit, interval: interval}
}

func (l *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(clientIP(r)) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *RateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket, ok := l.clients[key]
	if !ok || now.After(bucket.resetTime) {
		l.clients[key] = &clientBucket{count: 1, resetTime: now.Add(l.interval)}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		host, _, ok := strings.Cut(forwarded, ",")
		if ok {
			return strings.TrimSpace(host)
		}
		return strings.TrimSpace(forwarded)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
