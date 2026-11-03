package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

// IPRateLimiter provides token-bucket rate limiting per IP address to prevent brute force and DoS attacks.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientBucket
	rate    float64 // tokens added per second
	burst   int     // maximum token capacity
}

type clientBucket struct {
	tokens     float64
	lastUpdate time.Time
}

func NewIPRateLimiter(rate float64, burst int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientBucket),
		rate:    rate,
		burst:   burst,
	}

	// Periodically clean up stale client IPs
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.mu.Lock()
			now := time.Now()
			for ip, b := range limiter.clients {
				if now.Sub(b.lastUpdate) > 10*time.Minute {
					delete(limiter.clients, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

// Allow checks if the given IP has tokens remaining
func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, exists := l.clients[ip]
	if !exists {
		l.clients[ip] = &clientBucket{
			tokens:     float64(l.burst - 1),
			lastUpdate: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.lastUpdate = now
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	return false
}

// getClientIP extracts real client IP even when behind load balancers or reverse proxies
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// withRateLimit wraps an httprouter.Handle with token bucket rate limiting
func withRateLimit(limiter *IPRateLimiter, next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ip := getClientIP(r)
		if !limiter.Allow(ip) {
			http.Error(w, "Rate limit exceeded: too many requests. Please slow down.", http.StatusTooManyRequests)
			return
		}
		next(w, r, p)
	}
}
