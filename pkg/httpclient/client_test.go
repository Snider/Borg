package httpclient

import (
	"net/http"
	"net/url"
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Test case for no proxy
	t.Run("NoProxy", func(t *testing.T) {
		client, err := NewClient("", "", false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client.Transport != nil {
			t.Errorf("Expected nil Transport, got %T", client.Transport)
		}
	})

	// Test case for Tor
	t.Run("Tor", func(t *testing.T) {
		client, err := NewClient("", "", true)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("Expected http.Transport, got %T", client.Transport)
		}
		if transport.Dial == nil {
			t.Error("Expected a custom dialer for Tor, got nil")
		}
	})

	// Test case for a single proxy
	t.Run("SingleProxy", func(t *testing.T) {
		proxyURL := "http://localhost:8080"
		client, err := NewClient(proxyURL, "", false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("Expected http.Transport, got %T", client.Transport)
		}
		proxyFunc := transport.Proxy
		if proxyFunc == nil {
			t.Fatal("Expected a proxy function, got nil")
		}
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		proxy, err := proxyFunc(req)
		if err != nil {
			t.Fatalf("Expected no error from proxy func, got %v", err)
		}
		expectedProxy, _ := url.Parse(proxyURL)
		if proxy.String() != expectedProxy.String() {
			t.Errorf("Expected proxy %s, got %s", expectedProxy, proxy)
		}
	})

	// Test case for a proxy list
	t.Run("ProxyList", func(t *testing.T) {
		proxies := []string{
			"http://localhost:8081",
			"http://localhost:8082",
			"http://localhost:8083",
		}
		proxyFile, err := os.CreateTemp("", "proxies.txt")
		if err != nil {
			t.Fatalf("Failed to create temp proxy file: %v", err)
		}
		defer os.Remove(proxyFile.Name())
		for _, p := range proxies {
			proxyFile.WriteString(p + "\n")
		}
		proxyFile.Close()

		client, err := NewClient("", proxyFile.Name(), false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("Expected http.Transport, got %T", client.Transport)
		}
		proxyFunc := transport.Proxy
		if proxyFunc == nil {
			t.Fatal("Expected a proxy function, got nil")
		}
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		proxy, err := proxyFunc(req)
		if err != nil {
			t.Fatalf("Expected no error from proxy func, got %v", err)
		}

		found := false
		for _, p := range proxies {
			if proxy.String() == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected proxy to be one of %v, got %s", proxies, proxy)
		}
	})

	t.Run("SOCKS5WithAuth", func(t *testing.T) {
		proxyURL := "socks5://user:password@localhost:1080"
		client, err := NewClient(proxyURL, "", false)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("Expected http.Transport, got %T", client.Transport)
		}
		if transport.Dial == nil {
			t.Fatal("Expected a custom dialer for SOCKS5, got nil")
		}
	})
}
