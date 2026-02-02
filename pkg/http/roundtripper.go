package http

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitingRoundTripper is an http.RoundTripper that rate limits requests based on domain.
type RateLimitingRoundTripper struct {
	next     http.RoundTripper
	config   *Config
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
}

// NewRateLimitingRoundTripper creates a new RateLimitingRoundTripper.
func NewRateLimitingRoundTripper(config *Config, next http.RoundTripper) *RateLimitingRoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &RateLimitingRoundTripper{
		config:   config,
		next:     next,
		limiters: make(map[string]*rate.Limiter),
	}
}

func (r *RateLimitingRoundTripper) getLimiter(host string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := r.limiters[host]
	if !exists {
		rateLimit := r.config.GetRate(host)
		limiter = rate.NewLimiter(rate.Limit(rateLimit.RequestsPerSecond), rateLimit.Burst)
		r.limiters[host] = limiter
	}
	return limiter
}

// RoundTrip executes a single HTTP transaction, waiting for a token from the bucket first.
func (r *RateLimitingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	limiter := r.getLimiter(req.URL.Hostname())
	err := limiter.Wait(req.Context())
	if err != nil {
		return nil, err
	}

	resp, err := r.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		var delay time.Duration

		// Retry-After can be in seconds or an HTTP-date.
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			delay = time.Duration(seconds) * time.Second
		} else if t, err := http.ParseTime(retryAfter); err == nil {
			delay = time.Until(t)
		} else {
			// No valid Retry-After header, use a default backoff.
			delay = time.Second * 5
		}

		// Close the response body of the 429 response to allow the transport to reuse the connection.
		if resp.Body != nil {
			resp.Body.Close()
		}

		// Wait and retry the request once.
		time.Sleep(delay)
		return r.next.RoundTrip(req)
	}

	return resp, nil
}
