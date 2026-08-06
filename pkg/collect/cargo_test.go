package collect

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCargoCollector_Collect(t *testing.T) {
	client := &http.Client{
		Transport: &mockHTTPClient{
			responses: map[string]*http.Response{
				"https://crates.io/api/v1/crates/monero-rs": {
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"crate": {
							"name": "monero-rs"
						},
						"versions": [
							{
								"num": "0.1.0",
								"dl_path": "/api/v1/crates/monero-rs/0.1.0/download"
							}
						]
					}`)),
				},
				"https://crates.io/api/v1/crates/monero-rs/0.1.0/download": {
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("crate content"))),
				},
			},
		},
	}

	collector := &CargoCollector{client: client}
	dn, err := collector.Collect("monero-rs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := dn.Stat("metadata.json"); err != nil {
		t.Errorf("expected metadata.json to exist")
	}

	if _, err := dn.Stat("0.1.0.crate"); err != nil {
		t.Errorf("expected 0.1.0.crate to exist")
	}
}
