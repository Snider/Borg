package httpclient

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	totalTimeout := 10 * time.Second
	connectTimeout := 2 * time.Second
	tlsTimeout := 3 * time.Second
	headerTimeout := 5 * time.Second

	client := NewClient(totalTimeout, connectTimeout, tlsTimeout, headerTimeout)

	if client.Timeout != totalTimeout {
		t.Errorf("expected total timeout %v, got %v", totalTimeout, client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected client transport to be *http.Transport, got %T", client.Transport)
	}

	if transport.TLSHandshakeTimeout != tlsTimeout {
		t.Errorf("expected TLS handshake timeout %v, got %v", tlsTimeout, transport.TLSHandshakeTimeout)
	}

	if transport.ResponseHeaderTimeout != headerTimeout {
		t.Errorf("expected response header timeout %v, got %v", headerTimeout, transport.ResponseHeaderTimeout)
	}
}
