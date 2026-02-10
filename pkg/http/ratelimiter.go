package http

import (
	"context"
	"golang.org/x/time/rate"
)

// Limiter is a rate limiter that can be dynamically adjusted.
type Limiter struct {
	limiter *rate.Limiter
}

// NewLimiter creates a new Limiter.
func NewLimiter(r rate.Limit, b int) *Limiter {
	return &Limiter{
		limiter: rate.NewLimiter(r, b),
	}
}

// Wait waits for a token from the bucket.
func (l *Limiter) Wait(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

// SetLimit sets the rate limit.
func (l *Limiter) SetLimit(r rate.Limit) {
	l.limiter.SetLimit(r)
}
