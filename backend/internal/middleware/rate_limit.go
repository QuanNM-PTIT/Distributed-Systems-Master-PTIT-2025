package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientLimiter struct {
	tokens   float64
	lastSeen time.Time
}

type RateLimiter struct {
	rps       float64
	burst     float64
	clients   map[string]*clientLimiter
	mu        sync.Mutex
	cleanupCh chan struct{}
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rps:       rps,
		burst:     float64(burst),
		clients:   make(map[string]*clientLimiter),
		cleanupCh: make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (r *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			for key, cl := range r.clients {
				if time.Since(cl.lastSeen) > 10*time.Minute {
					delete(r.clients, key)
				}
			}
			r.mu.Unlock()
		case <-r.cleanupCh:
			return
		}
	}
}

func (r *RateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cl, ok := r.clients[key]
	if !ok {
		r.clients[key] = &clientLimiter{tokens: r.burst - 1, lastSeen: now}
		return true
	}
	elapsed := now.Sub(cl.lastSeen).Seconds()
	cl.tokens = minFloat(r.burst, cl.tokens+elapsed*r.rps)
	cl.lastSeen = now
	if cl.tokens < 1 {
		return false
	}
	cl.tokens -= 1
	return true
}

func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
