package mocks

import (
	"bytes"
	"io"
	"net/http"
)

// MockRoundTripper is a mock implementation of http.RoundTripper.
type MockRoundTripper struct {
	Responses map[string]*http.Response
}

// RoundTrip implements the http.RoundTripper interface.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	if resp, ok := m.Responses[url]; ok {
		// Create a new reader for the body each time, as it can be read only once.
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close() // close original body
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return resp, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
		Header:     make(http.Header),
	}, nil
}

// NewMockClient creates a new http.Client with a MockRoundTripper.
func NewMockClient(responses map[string]*http.Response) *http.Client {
	return &http.Client{
		Transport: &MockRoundTripper{
			Responses: responses,
		},
	}
}
