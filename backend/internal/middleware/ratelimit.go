package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/thesct22/ezyshare/backend/internal/telemetry"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu      sync.RWMutex
	ips     map[string]*ipLimiter
	rate    rate.Limit
	burst   int
	metrics *telemetry.Metrics
	quit    chan struct{}
}

func NewIPRateLimiter(r rate.Limit, b int, metrics *telemetry.Metrics) *IPRateLimiter {
	lim := &IPRateLimiter{
		ips:     make(map[string]*ipLimiter),
		rate:    r,
		burst:   b,
		metrics: metrics,
		quit:    make(chan struct{}),
	}

	go lim.cleanupLoop()
	return lim
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	lim, exists := i.ips[ip]
	if !exists {
		l := rate.NewLimiter(i.rate, i.burst)
		i.ips[ip] = &ipLimiter{limiter: l, lastSeen: time.Now()}
		return l
	}

	lim.lastSeen = time.Now()
	return lim.limiter
}

func (i *IPRateLimiter) Allow(ip string) bool {
	return i.GetLimiter(ip).Allow()
}

func (i *IPRateLimiter) Stop() {
	close(i.quit)
}

func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			i.mu.Lock()
			for ip, entry := range i.ips {
				if time.Since(entry.lastSeen) > 10*time.Minute {
					delete(i.ips, ip)
				}
			}
			i.mu.Unlock()
		case <-i.quit:
			return
		}
	}
}

func RateLimitMiddleware(limiter *IPRateLimiter, endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := telemetry.GetClientIP(r)

			if !limiter.Allow(ip) {
				if limiter.metrics != nil {
					limiter.metrics.RateLimitExceeded.WithLabelValues(endpoint).Inc()
				}
				w.Header().Set("Retry-After", "60")
				http.Error(w, "429 Too Many Requests - Rate Limit Exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
