package server

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter to protect server from spam and abuse.
type RateLimiter struct {
	rate       float64 // tokens per second
	capacity   float64 // max tokens
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new token bucket rate limiter.
// e.g. rate = 3.0 messages per second, burst capacity = 8.0 messages.
func NewRateLimiter(rate, capacity float64) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastUpdate: time.Now(),
	}
}

// Allow returns true if the action is allowed under the rate limit, or false if limited.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	rl.lastUpdate = now

	// Refill tokens
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}
