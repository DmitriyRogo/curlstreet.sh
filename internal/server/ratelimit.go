package server

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu         sync.Mutex
	limiters   map[string]*ipLimiter
	r          rate.Limit
	burst      int
	trustedNet *net.IPNet // nil = no trusted proxy; when set, X-Forwarded-For is used for IPs in this CIDR
}

func newRateLimiter(requestsPerMinute, burst int, trustedNet *net.IPNet) *rateLimiter {
	rl := &rateLimiter{
		limiters:   make(map[string]*ipLimiter),
		r:          rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:      burst,
		trustedNet: trustedNet,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.limiters[ip]
	if !ok {
		entry = &ipLimiter{limiter: rate.NewLimiter(rl.r, rl.burst)}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanup removes limiters for IPs not seen in the last 5 minutes.
func (rl *rateLimiter) cleanup() {
	for range time.Tick(2 * time.Minute) {
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, rl.trustedNet)
		if !rl.get(ip).Allow() {
			http.Error(w, "rate limit exceeded — slow down", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
