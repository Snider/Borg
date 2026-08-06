package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync/atomic"
	"time"
)

// Metrics holds the connection reuse metrics.
type Metrics struct {
	ConnectionsReused  int64
	ConnectionsCreated int64
}

// Options represents the configuration for the HTTP client.
type Options struct {
	MaxPerHost  int
	MaxIdle     int
	IdleTimeout time.Duration
	NoKeepAlive bool
	HTTP1       bool
}

// New creates a new HTTP client with the given options.
func New(opts Options) (*http.Client, *Metrics) {
	metrics := &Metrics{}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        opts.MaxIdle,
		IdleConnTimeout:     opts.IdleTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxConnsPerHost:     opts.MaxPerHost,
		DisableKeepAlives:   opts.NoKeepAlive,
	}

	if opts.HTTP1 {
		// Disable HTTP/2 by preventing the TLS next protocol negotiation.
		transport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	}

	return &http.Client{
		Transport: &metricsRoundTripper{
			transport: transport,
			metrics:   metrics,
		},
	}, metrics
}

// metricsRoundTripper is a custom http.RoundTripper that collects connection reuse metrics.
type metricsRoundTripper struct {
	transport http.RoundTripper
	metrics   *Metrics
}

func (m *metricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				atomic.AddInt64(&m.metrics.ConnectionsReused, 1)
			} else {
				atomic.AddInt64(&m.metrics.ConnectionsCreated, 1)
			}
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	return m.transport.RoundTrip(req)
}
