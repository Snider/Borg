package collect

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockGoHTTPClient struct {
	responses map[string]*http.Response
}

func (c *mockGoHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.responses[req.URL.String()], nil
}

func TestGoCollector_Collect(t *testing.T) {
	client := &http.Client{
		Transport: &mockGoHTTPClient{
			responses: map[string]*http.Response{
				"https://proxy.golang.org/github.com/monero-ecosystem/go-monero/@v/list": {
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("v0.1.0\nv0.2.0")),
				},
				"https://proxy.golang.org/github.com/monero-ecosystem/go-monero/@v/v0.1.0.zip": {
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("zip content v0.1.0"))),
				},
				"https://proxy.golang.org/github.com/monero-ecosystem/go-monero/@v/v0.2.0.zip": {
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("zip content v0.2.0"))),
				},
			},
		},
	}

	collector := &GoCollector{client: client}
	dn, err := collector.Collect("github.com/monero-ecosystem/go-monero")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := dn.Stat("v0.1.0.zip"); err != nil {
		t.Errorf("expected v0.1.0.zip to exist")
	}

	if _, err := dn.Stat("v0.2.0.zip"); err != nil {
		t.Errorf("expected v0.2.0.zip to exist")
	}
}
