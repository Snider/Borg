package mocks

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// MockRoundTripper is a mock implementation of http.RoundTripper.
type MockRoundTripper struct {
	mu        sync.RWMutex
	responses map[string]*http.Response
}

// RoundTrip implements the http.RoundTripper interface.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	m.mu.RLock()
	resp, ok := m.responses[url]
	m.mu.RUnlock()

	if ok {
		// Read the original body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close() // close original body

		// Re-hydrate the original body so it can be read again
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Create a deep copy of the response
		newResp := &http.Response{
			Status:           resp.Status,
			StatusCode:       resp.StatusCode,
			Proto:            resp.Proto,
			ProtoMajor:       resp.ProtoMajor,
			ProtoMinor:       resp.ProtoMinor,
			Header:           resp.Header.Clone(),
			Body:             io.NopCloser(bytes.NewReader(bodyBytes)),
			ContentLength:    resp.ContentLength,
			TransferEncoding: resp.TransferEncoding,
			Close:            resp.Close,
			Uncompressed:     resp.Uncompressed,
			Trailer:          resp.Trailer.Clone(),
			Request:          resp.Request,
			TLS:              resp.TLS,
		}
		return newResp, nil
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
			responses: responses,
		},
	}
}
