package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

// RateLimiter is a per-client-IP token bucket. Idle entries are swept lazily (no background goroutine).
type RateLimiter struct {
	limit rate.Limit
	burst int

	mu        sync.Mutex
	clients   map[string]*client
	lastSweep time.Time
}

type client struct {
	lim  *rate.Limiter
	seen time.Time
}

// NewRateLimiter allows `perMinute` requests per minute with the given burst.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	return &RateLimiter{
		limit:   rate.Limit(float64(perMinute) / 60),
		burst:   burst,
		clients: make(map[string]*client),
	}
}

func (rl *RateLimiter) allow(key string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if now.Sub(rl.lastSweep) > time.Minute {
		for k, c := range rl.clients {
			if now.Sub(c.seen) > 10*time.Minute {
				delete(rl.clients, k)
			}
		}
		rl.lastSweep = now
	}
	c, ok := rl.clients[key]
	if !ok {
		c = &client{lim: rate.NewLimiter(rl.limit, rl.burst)}
		rl.clients[key] = c
	}
	c.seen = now
	return c.lim.AllowN(now, 1)
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ClientIP honours X-Forwarded-For only from the trusted loopback proxy.
		if !rl.allow(c.ClientIP(), time.Now()) {
			c.Header("Retry-After", "60")
			httpx.Abort(c, http.StatusTooManyRequests, "rate_limited", "Too many requests, please try again later")
			return
		}
		c.Next()
	}
}
