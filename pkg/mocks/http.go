package mocks

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// MockRoundTripper is a mock implementation of http.RoundTripper.
type MockRoundTripper struct {
	responses map[string]*http.Response
}

// RoundTrip implements the http.RoundTripper interface.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if res, ok := m.responses[req.URL.String()]; ok {
		// Make a copy of the response so it can be read multiple times
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		return res, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       ioutil.NopCloser(bytes.NewBufferString("not found")),
	}, nil
}

// NewMockClient returns a mock HTTP client that returns the given responses.
func NewMockClient(responses map[string]*http.Response) *http.Client {
	return &http.Client{
		Transport: &MockRoundTripper{
			responses: responses,
		},
	}
}
