package httpclient

import (
	"net"
	"net/http"
	"time"
)

// NewClient creates a new http.Client with configurable timeouts.
func NewClient(totalTimeout, connectTimeout, tlsTimeout, headerTimeout time.Duration) *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: connectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   tlsTimeout,
		ResponseHeaderTimeout: headerTimeout,
	}

	return &http.Client{
		Timeout:   totalTimeout,
		Transport: transport,
	}
}
