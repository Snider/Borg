// Package mocks provides mock implementations of interfaces for testing purposes.
package mocks

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// MockRoundTripper is a mock implementation of the http.RoundTripper interface,
// used for mocking HTTP clients in tests. It allows setting predefined responses
// for specific URLs.
type MockRoundTripper struct {
	mu        sync.RWMutex
	responses map[string]*http.Response
}

// SetResponses sets the mock responses for the MockRoundTripper in a thread-safe
// manner. The responses map keys are URLs and values are the http.Response
// objects to be returned for those URLs.
func (m *MockRoundTripper) SetResponses(responses map[string]*http.Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = responses
}

// RoundTrip is the implementation of the http.RoundTripper interface. It looks
// up the request URL in the mock responses map and returns the corresponding
// response. If no response is found, it returns a 404 Not Found response.
// It performs a deep copy of the response to prevent race conditions on the
// response body.
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

// NewMockClient creates a new http.Client that uses the MockRoundTripper. This
// is a convenience function for creating a mock HTTP client for tests. The
// responses map is defensively copied to prevent race conditions.
//
// Example:
//
//	mockResponses := map[string]*http.Response{
//		"https://example.com": {
//			StatusCode: http.StatusOK,
//			Body:       io.NopCloser(bytes.NewBufferString("Hello")),
//		},
//	}
//	client := mocks.NewMockClient(mockResponses)
//	resp, err := client.Get("https://example.com")
//	// ...
func NewMockClient(responses map[string]*http.Response) *http.Client {
	responsesCopy := make(map[string]*http.Response)
	if responses != nil {
		for k, v := range responses {
			responsesCopy[k] = v
		}
	}
	return &http.Client{
		Transport: &MockRoundTripper{
			responses: responsesCopy,
		},
	}
}
