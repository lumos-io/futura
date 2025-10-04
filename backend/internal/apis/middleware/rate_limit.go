package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/backend/internal/shared/utils"
)

// RateLimiter tracks request counts per IP
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // requests allowed
	window   time.Duration // time window
}

// Visitor tracks requests for a single IP
type Visitor struct {
	count      int
	lastReset  time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per window
// window: time window duration (e.g., 1 * time.Minute)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup old visitors every 5 minutes
	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, visitor := range rl.visitors {
		visitor.mu.Lock()
		if now.Sub(visitor.lastReset) > rl.window*2 {
			delete(rl.visitors, ip)
		}
		visitor.mu.Unlock()
	}
}

func (rl *RateLimiter) getVisitor(ip string) *Visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	visitor, exists := rl.visitors[ip]
	if !exists {
		visitor = &Visitor{
			count:     0,
			lastReset: time.Now(),
		}
		rl.visitors[ip] = visitor
	}

	return visitor
}

func (rl *RateLimiter) isAllowed(ip string) bool {
	visitor := rl.getVisitor(ip)

	visitor.mu.Lock()
	defer visitor.mu.Unlock()

	now := time.Now()

	// Reset counter if window has passed
	if now.Sub(visitor.lastReset) > rl.window {
		visitor.count = 0
		visitor.lastReset = now
	}

	// Check if limit exceeded
	if visitor.count >= rl.rate {
		return false
	}

	visitor.count++
	return true
}

// Middleware returns a Gin middleware function for rate limiting
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.isAllowed(ip) {
			utils.RespondError(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RefreshTokenRateLimiter is a pre-configured rate limiter for refresh token endpoint
// Allows 10 requests per minute per IP
var RefreshTokenRateLimiter = NewRateLimiter(10, 1*time.Minute)

// OAuthCallbackRateLimiter is a pre-configured rate limiter for OAuth callbacks
// Allows 5 requests per minute per IP
var OAuthCallbackRateLimiter = NewRateLimiter(5, 1*time.Minute)
