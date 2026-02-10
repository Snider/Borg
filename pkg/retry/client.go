package retry

import (
	"math/rand"
	"net/http"
	"time"
)

// Backoff is a time.Duration that represents the backoff strategy.
type Backoff time.Duration

// Exponential returns a new Backoff duration that is the current duration
// multiplied by 2.
func (b Backoff) Exponential() Backoff {
	return Backoff(time.Duration(b) * 2)
}

// Jitter returns a new Backoff duration with a random jitter added.
func (b Backoff) Jitter(factor float64) Backoff {
	if factor <= 0 {
		return b
	}
	jitter := time.Duration(rand.Float64() * factor * float64(b))
	return Backoff(time.Duration(b) + jitter)
}

// Transport is an http.RoundTripper that automatically retries requests.
type Transport struct {
	// Transport is the underlying http.RoundTripper to use for requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// Retries is the maximum number of retries to attempt.
	Retries int

	// InitialBackoff is the initial backoff duration.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	MaxBackoff time.Duration

	// Jitter is the jitter factor to apply to backoff durations.
	Jitter float64
}

// NewTransport creates a new Transport with default values.
func NewTransport() *Transport {
	return &Transport{
		Transport:      http.DefaultTransport,
		Retries:        3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Jitter:         0.1,
	}
}

// RoundTrip implements the http.RoundTripper interface.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	var backoff = Backoff(t.InitialBackoff)

	for i := 0; i < t.Retries; i++ {
		resp, err = t.transport().RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, err
		}

		if i < t.Retries-1 {
			time.Sleep(time.Duration(backoff))
			backoff = backoff.Exponential().Jitter(t.Jitter)
			if backoff > Backoff(t.MaxBackoff) {
				backoff = Backoff(t.MaxBackoff)
			}
		}
	}
	return resp, err
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

// NewClient returns a new http.Client that uses the Transport.
func NewClient(transport *Transport) *http.Client {
	return &http.Client{
		Transport: transport,
	}
}

var (
	// DefaultTransport is the default transport with retry logic.
	DefaultTransport = NewTransport()
	// DefaultClient is the default client that uses the default transport.
	DefaultClient = NewClient(DefaultTransport)
)
