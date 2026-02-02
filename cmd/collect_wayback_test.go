package cmd

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// MockRoundTripper is a mock implementation of http.RoundTripper for testing.
type MockRoundTripper struct {
	Response      *http.Response
	Err           error
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req)
	}
	return m.Response, m.Err
}

func NewMockClient(responseBody string, statusCode int) *http.Client {
	return &http.Client{
		Transport: &MockRoundTripper{
			Response: &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
			},
		},
	}
}

func TestWaybackList(t *testing.T) {
	t.Cleanup(func() {
		RootCmd.SetArgs([]string{})
	})
	mockResponse := `[
		["urlkey","timestamp","original","mimetype","statuscode","digest","length"],
		["com,example)/", "20220101000000", "http://example.com/", "text/html", "200", "DIGEST", "1234"]
	]`
	http.DefaultClient = NewMockClient(mockResponse, http.StatusOK)

	output, err := executeCommand(RootCmd, "wayback", "list", "http://example.com")
	if err != nil {
		t.Fatalf("executeCommand returned an unexpected error: %v", err)
	}

	if !strings.Contains(output, "20220101000000") {
		t.Errorf("Expected output to contain timestamp '20220101000000', got '%s'", output)
	}
}

func TestWaybackCollect(t *testing.T) {
	t.Cleanup(func() {
		RootCmd.SetArgs([]string{})
	})
	t.Run("Good - Latest with Assets", func(t *testing.T) {
		mockListResponse := `[
			["urlkey","timestamp","original","mimetype","statuscode","digest","length"],
			["com,example)/", "20230101000000", "http://example.com/", "text/html", "200", "DIGEST1", "1234"]
		]`
		mockAssetsResponse := `[
			["urlkey","timestamp","original","mimetype","statuscode","digest","length"],
			["com,example)/", "20230101000000", "http://example.com/", "text/html", "200", "DIGEST1", "1234"],
			["com,example)/css/style.css", "20230101000000", "http://example.com/css/style.css", "text/css", "200", "DIGEST2", "5678"]
		]`
		mockHTMLContent := "<html><head><link rel='stylesheet' href='/css/style.css'></head><body>Hello</body></html>"
		mockCSSContent := "body { color: red; }"

		// This is still a simplified mock, but it's better.
		// A more robust solution would use a mock server or a more sophisticated RoundTripper.
		var requestCount int
		http.DefaultClient = &http.Client{
			Transport: &MockRoundTripper{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("")), // Placeholder
				},
			},
		}
		http.DefaultClient.Transport.(*MockRoundTripper).Response.Body = io.NopCloser(bytes.NewBufferString(mockListResponse))
		http.DefaultClient.Transport.(*MockRoundTripper).RoundTripFunc = func(req *http.Request) (*http.Response, error) {
			var body string
			if requestCount == 0 {
				body = mockListResponse
			} else if requestCount == 1 {
				body = mockAssetsResponse
			} else if strings.Contains(req.URL.Path, "style.css") {
				body = mockCSSContent
			} else {
				body = mockHTMLContent
			}
			requestCount++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}

		tempDir, err := os.MkdirTemp("", "borg-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		_, err = executeCommand(RootCmd, "wayback", "collect", "http://example.com", "--latest", "--output", tempDir)
		if err != nil {
			t.Fatalf("executeCommand returned an unexpected error: %v", err)
		}

		// Verify TIMELINE.md
		timelineFile := tempDir + "/TIMELINE.md"
		if _, err := os.Stat(timelineFile); os.IsNotExist(err) {
			t.Errorf("Expected TIMELINE.md to be created in %s", tempDir)
		}

		// Verify index.html
		indexFile := tempDir + "/20230101000000/index.html"
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			t.Fatalf("Expected index.html to be created in %s", indexFile)
		}
		content, err := os.ReadFile(indexFile)
		if err != nil {
			t.Fatalf("Failed to read index.html: %v", err)
		}
		if !strings.Contains(string(content), "Hello") {
			t.Errorf("index.html content is incorrect")
		}

		// Verify style.css
		cssFile := tempDir + "/20230101000000/css/style.css"
		if _, err := os.Stat(cssFile); os.IsNotExist(err) {
			t.Fatalf("Expected style.css to be created in %s", cssFile)
		}
		content, err = os.ReadFile(cssFile)
		if err != nil {
			t.Fatalf("Failed to read style.css: %v", err)
		}
		if !strings.Contains(string(content), "color: red") {
			t.Errorf("style.css content is incorrect")
		}
	})
}
