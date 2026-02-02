package collect

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockHTTPClient struct {
	responses map[string]*http.Response
}

func (c *mockHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.responses[req.URL.String()], nil
}

func TestNPMCollector_Collect(t *testing.T) {
	client := &http.Client{
		Transport: &mockHTTPClient{
			responses: map[string]*http.Response{
				"https://registry.npmjs.org/@monero-project/monero-ts": {
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"name": "@monero-project/monero-ts",
						"versions": {
							"1.0.0": {
								"dist": {
									"tarball": "https://registry.npmjs.org/@monero-project/monero-ts/-/monero-ts-1.0.0.tgz"
								}
							}
						}
					}`)),
				},
				"https://registry.npmjs.org/@monero-project/monero-ts/-/monero-ts-1.0.0.tgz": {
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("tarball content"))),
				},
			},
		},
	}

	collector := &NPMCollector{client: client}
	dn, err := collector.Collect("@monero-project/monero-ts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := dn.Stat("metadata.json"); err != nil {
		t.Errorf("expected metadata.json to exist")
	}

	if _, err := dn.Stat("1.0.0.tgz"); err != nil {
		t.Errorf("expected 1.0.0.tgz to exist")
	}
}
