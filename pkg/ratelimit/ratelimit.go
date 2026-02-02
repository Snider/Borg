package ratelimit

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Limiter is a simple token bucket rate limiter.
type Limiter struct {
	c chan time.Time
}

// NewLimiter creates a new Limiter.
func NewLimiter(rate int64, per time.Duration) *Limiter {
	l := &Limiter{
		c: make(chan time.Time, rate),
	}
	go func() {
		ticker := time.NewTicker(per / time.Duration(rate))
		defer ticker.Stop()
		for t := range ticker.C {
			select {
			case l.c <- t:
			default:
			}
		}
	}()
	return l
}

// Wait blocks until a token is available.
func (l *Limiter) Wait() {
	<-l.c
}

// rateLimitedRoundTripper is an http.RoundTripper that limits the bandwidth.
type rateLimitedRoundTripper struct {
	transport http.RoundTripper
	limiter   *Limiter
}

// NewRateLimitedRoundTripper creates a new rateLimitedRoundTripper.
func NewRateLimitedRoundTripper(transport http.RoundTripper, bytesPerSec int64) http.RoundTripper {
	if bytesPerSec <= 0 {
		return transport
	}
	return &rateLimitedRoundTripper{
		transport: transport,
		limiter:   NewLimiter(bytesPerSec, time.Second),
	}
}

// RoundTrip implements the http.RoundTripper interface.
func (t *rateLimitedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	resp.Body = &rateLimitedResponseBody{
		body:    resp.Body,
		limiter: t.limiter,
	}

	return resp, nil
}

// rateLimitedResponseBody is an io.ReadCloser that limits the bandwidth.
type rateLimitedResponseBody struct {
	body    io.ReadCloser
	limiter *Limiter
}

// Read implements the io.Reader interface.
func (b *rateLimitedResponseBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if err != nil {
		return n, err
	}
	for i := 0; i < n; i++ {
		b.limiter.Wait()
	}
	return n, nil
}

// Close implements the io.Closer interface.
func (b *rateLimitedResponseBody) Close() error {
	return b.body.Close()
}

// ParseBandwidth parses a human-readable bandwidth string (e.g., "1MB/s")
// and returns the equivalent in bytes per second.
func ParseBandwidth(bandwidth string) (int64, error) {
	if bandwidth == "" || bandwidth == "0" {
		return 0, nil
	}

	re := regexp.MustCompile(`(?i)^(\d+)\s*(KB/s|MB/s|Mbps)$`)
	matches := re.FindStringSubmatch(bandwidth)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid bandwidth format: %s", bandwidth)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid bandwidth value: %s", matches[1])
	}

	unit := strings.ToUpper(matches[2])
	switch unit {
	case "KB/S":
		return value * 1024, nil
	case "MB/S":
		return value * 1024 * 1024, nil
	case "MBPS":
		return value * 1024 * 1024 / 8, nil
	default:
		return 0, fmt.Errorf("unknown bandwidth unit: %s", unit)
	}
}
