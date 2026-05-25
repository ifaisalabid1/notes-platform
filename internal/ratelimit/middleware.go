package ratelimit

import "net/http"

func Middleware(limiter *IPLimiter, tooManyRequests http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(r) {
				tooManyRequests(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
