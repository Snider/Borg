package ratelimit

import (
	"testing"
	"time"
)

func TestParseBandwidth(t *testing.T) {
	testCases := []struct {
		input    string
		expected int64
		err      bool
	}{
		{"1KB/s", 1024, false},
		{"1MB/s", 1024 * 1024, false},
		{"1Mbps", 1024 * 1024 / 8, false},
		{"500KB/s", 500 * 1024, false},
		{"10MB/s", 10 * 1024 * 1024, false},
		{"8Mbps", 1024 * 1024, false},
		{"0", 0, false},
		{"", 0, false},
		{"1 GB/s", 0, true},
		{"1MB", 0, true},
		{"MB/s", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual, err := ParseBandwidth(tc.input)
			if (err != nil) != tc.err {
				t.Errorf("expected error: %v, got: %v", tc.err, err)
			}
			if actual != tc.expected {
				t.Errorf("expected: %d, got: %d", tc.expected, actual)
			}
		})
	}
}

func TestLimiter(t *testing.T) {
	// Test case 1: 10 tokens per second
	limiter1 := NewLimiter(10, time.Second)
	start1 := time.Now()
	for i := 0; i < 10; i++ {
		limiter1.Wait()
	}
	elapsed1 := time.Since(start1)
	if elapsed1 < 900*time.Millisecond || elapsed1 > 1100*time.Millisecond {
		t.Errorf("expected to take around 1s for 10 tokens at 10 tokens/sec, but took %s", elapsed1)
	}

	// Test case 2: 100 tokens per second
	limiter2 := NewLimiter(100, time.Second)
	start2 := time.Now()
	for i := 0; i < 10; i++ {
		limiter2.Wait()
	}
	elapsed2 := time.Since(start2)
	if elapsed2 < 90*time.Millisecond || elapsed2 > 110*time.Millisecond {
		t.Errorf("expected to take around 100ms for 10 tokens at 100 tokens/sec, but took %s", elapsed2)
	}
}
