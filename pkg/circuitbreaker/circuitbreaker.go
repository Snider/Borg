
package circuitbreaker

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// State represents the state of the circuit breaker.
type State int

const (
	// ClosedState is the initial state of the circuit breaker.
	ClosedState State = iota
	// OpenState is the state when the circuit breaker is tripped.
	OpenState
	// HalfOpenState is the state when the circuit breaker is testing for recovery.
	HalfOpenState
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case ClosedState:
		return "CLOSED"
	case OpenState:
		return "OPEN"
	case HalfOpenState:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

// Settings configures the circuit breaker.
type Settings struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes to close the circuit.
	SuccessThreshold int
	// Cooldown is the time to wait in the open state before transitioning to half-open.
	Cooldown time.Duration
	// HalfOpenRequests is the number of test requests to allow in the half-open state.
	HalfOpenRequests int
	// Logger is the logger to use for state changes.
	Logger *slog.Logger
}

// CircuitBreaker is a state machine that prevents repeated calls to a failing service.
type CircuitBreaker struct {
	settings         Settings
	domain           string
	mu               sync.Mutex
	state            State
	failures         int
	successes        int
	halfOpenRequests int
	lastError        error
	expiry           time.Time
}

// New creates a new CircuitBreaker.
func New(domain string, settings Settings) *CircuitBreaker {
	if settings.Logger == nil {
		settings.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
	}
	return &CircuitBreaker{
		settings: settings,
		domain:   domain,
		state:    ClosedState,
	}
}

// Execute runs the given function, protected by the circuit breaker.
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case OpenState:
		if time.Now().After(cb.expiry) {
			cb.setState(HalfOpenState)
			return cb.executeHalfOpen(fn)
		}
		return nil, fmt.Errorf("circuit is open for: %w", cb.lastError)
	case HalfOpenState:
		return cb.executeHalfOpen(fn)
	default: // ClosedState
		return cb.executeClosed(fn)
	}
}

func (cb *CircuitBreaker) executeClosed(fn func() (interface{}, error)) (interface{}, error) {
	res, err := fn()
	if err != nil {
		cb.failures++
		if cb.failures >= cb.settings.FailureThreshold {
			cb.lastError = err
			cb.setState(OpenState)
		}
		return nil, err
	}

	cb.failures = 0
	return res, nil
}

func (cb *CircuitBreaker) executeHalfOpen(fn func() (interface{}, error)) (interface{}, error) {
	if cb.halfOpenRequests >= cb.settings.HalfOpenRequests {
		return nil, fmt.Errorf("circuit is half-open, test requests exhausted: %w", cb.lastError)
	}
	cb.halfOpenRequests++

	res, err := fn()
	if err != nil {
		cb.lastError = err
		cb.setState(OpenState)
		return nil, err
	}

	cb.successes++
	// If enough test requests succeed, close the circuit.
	if cb.successes >= cb.settings.SuccessThreshold {
		cb.setState(ClosedState)
	}

	return res, nil
}

func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	cb.state = state
	logMessage := fmt.Sprintf("Circuit %s for %s", state, cb.domain)
	if state == OpenState {
		cb.settings.Logger.Warn(logMessage)
	} else {
		cb.settings.Logger.Info(logMessage)
	}

	switch state {
	case OpenState:
		cb.expiry = time.Now().Add(cb.settings.Cooldown)
		cb.successes = 0
	case ClosedState:
		cb.failures = 0
		cb.successes = 0
	case HalfOpenState:
		cb.failures = 0
		cb.halfOpenRequests = 0
	}
}
