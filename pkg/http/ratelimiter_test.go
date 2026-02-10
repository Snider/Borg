package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	limiter := NewLimiter(rate.Limit(10), 1)
	start := time.Now()
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		limiter.Wait(ctx)
	}
	elapsed := time.Since(start)
	// Loosen the timing constraint slightly to avoid flakes in CI
	if elapsed > 1*time.Second {
		t.Errorf("Rate limiter is slower than expected: %v", elapsed)
	}
}

func TestConfigParsing(t *testing.T) {
	config, err := ParseConfig("testdata/.borg-rates.yaml")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if config.Defaults.RequestsPerSecond != 1 {
		t.Errorf("Expected default requests per second to be 1, got %v", config.Defaults.RequestsPerSecond)
	}

	if config.Defaults.Burst != 5 {
		t.Errorf("Expected default burst to be 5, got %v", config.Defaults.Burst)
	}

	githubRate := config.GetRate("api.github.com")
	if githubRate.RequestsPerSecond != 0.5 {
		t.Errorf("Expected api.github.com requests per second to be 0.5, got %v", githubRate.RequestsPerSecond)
	}

	archiveRate := config.GetRate("subdomain.archive.org")
	if archiveRate.RequestsPerSecond != 1 {
		t.Errorf("Expected subdomain.archive.org requests per second to be 1, got %v", archiveRate.RequestsPerSecond)
	}
}

func TestRateLimitingRoundTripper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Defaults: Rate{
			RequestsPerSecond: 100,
			Burst:             1,
		},
	}
	transport := NewRateLimitingRoundTripper(config, http.DefaultTransport)
	client := &http.Client{Transport: transport}

	start := time.Now()
	for i := 0; i < 10; i++ {
		_, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Rate limiter is slower than expected: %v", elapsed)
	}
}

func TestRateLimitingRoundTripper_429(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		Defaults: Rate{
			RequestsPerSecond: 100,
			Burst:             1,
		},
	}
	transport := NewRateLimitingRoundTripper(config, http.DefaultTransport)
	client := &http.Client{Transport: transport}

	start := time.Now()
	_, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < 1*time.Second {
		t.Errorf("Expected to wait at least 1 second, but waited %v", elapsed)
	}
	if requests != 2 {
		t.Errorf("Expected 2 requests, but got %d", requests)
	}
}
