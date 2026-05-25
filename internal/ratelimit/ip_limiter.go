package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type IPLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewIPLimiter(r rate.Limit, burst int, ttl time.Duration) *IPLimiter {
	limiter := &IPLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    burst,
		ttl:      ttl,
	}

	go limiter.cleanupLoop()

	return limiter
}

func (l *IPLimiter) Allow(r *http.Request) bool {
	ip := clientIP(r)
	if ip == "" {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	item, exists := l.visitors[ip]
	if !exists {
		item = &visitor{
			limiter: rate.NewLimiter(l.rate, l.burst),
		}

		l.visitors[ip] = item
	}

	item.lastSeen = time.Now()

	return item.limiter.Allow()
}

func (l *IPLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

func (l *IPLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	for ip, item := range l.visitors {
		if now.Sub(item.lastSeen) > l.ttl {
			delete(l.visitors, ip)
		}
	}
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")

		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				return parsedIP.String()
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if parsedIP := net.ParseIP(host); parsedIP != nil {
			return parsedIP.String()
		}
	}

	if parsedIP := net.ParseIP(r.RemoteAddr); parsedIP != nil {
		return parsedIP.String()
	}

	return ""
}
