
package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker(t *testing.T) {
	settings := Settings{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Cooldown:         100 * time.Millisecond,
		HalfOpenRequests: 2,
	}
	cb := New("test.com", settings)

	// Initially closed
	_, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}

	// Trip the breaker
	_, err = cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure 1")
	})
	if err == nil {
		t.Error("Expected failure, got nil")
	}
	_, err = cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure 2")
	})
	if err == nil {
		t.Error("Expected failure, got nil")
	}

	// Now open
	_, err = cb.Execute(func() (interface{}, error) {
		return "should not be called", nil
	})
	if err == nil || err.Error() != "circuit is open for: failure 2" {
		t.Errorf("Expected open circuit error, got %v", err)
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	// Half-open, should succeed
	_, err = cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if err != nil {
		t.Errorf("Expected success in half-open, got %v", err)
	}

	// Still half-open, need another success
	_, err = cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if err != nil {
		t.Errorf("Expected success in half-open, got %v", err)
	}

	// Now closed again
	_, err = cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	if err != nil {
		t.Errorf("Expected success in closed state, got %v", err)
	}

	// Trip again to test half-open failure
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure 1")
	})
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure 2")
	})

	time.Sleep(150 * time.Millisecond)

	// Half-open, but fail
	_, err = cb.Execute(func() (interface{}, error) {
		return nil, errors.New("half-open failure")
	})
	if err == nil {
		t.Error("Expected failure in half-open, got nil")
	}

	// Should be open again
	_, err = cb.Execute(func() (interface{}, error) {
		return "should not be called", nil
	})
	if err == nil || err.Error() != "circuit is open for: half-open failure" {
		t.Errorf("Expected open circuit error, got %v", err)
	}
}
