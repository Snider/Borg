package wayback

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// MockRoundTripper is a mock implementation of http.RoundTripper for testing.
type MockRoundTripper struct {
	Response *http.Response
	Err      error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
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

func TestListSnapshots(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		mockResponse := `[
			["urlkey","timestamp","original","mimetype","statuscode","digest","length"],
			["com,example)/", "20220101000000", "http://example.com/", "text/html", "200", "DIGEST", "1234"],
			["com,example)/", "20230101000000", "http://example.com/", "text/html", "200", "DIGEST", "5678"]
		]`
		http.DefaultClient = NewMockClient(mockResponse, http.StatusOK)

		snapshots, err := ListSnapshots("http://example.com")
		if err != nil {
			t.Fatalf("ListSnapshots returned an unexpected error: %v", err)
		}
		if len(snapshots) != 2 {
			t.Fatalf("Expected 2 snapshots, got %d", len(snapshots))
		}
		if snapshots[0].Timestamp != "20220101000000" {
			t.Errorf("Expected timestamp '20220101000000', got '%s'", snapshots[0].Timestamp)
		}
	})

	t.Run("Bad - API error", func(t *testing.T) {
		http.DefaultClient = NewMockClient("server error", http.StatusInternalServerError)
		_, err := ListSnapshots("http://example.com")
		if err == nil {
			t.Fatal("ListSnapshots did not return an error for a non-200 response")
		}
	})

	t.Run("Ugly - Malformed JSON", func(t *testing.T) {
		http.DefaultClient = NewMockClient(`[`, http.StatusOK)
		_, err := ListSnapshots("http://example.com")
		if err == nil {
			t.Fatal("ListSnapshots did not return an error for malformed JSON")
		}
	})
}

func TestDownloadSnapshot(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		mockResponse := "<html><body>Hello, World!</body></html>"
		http.DefaultClient = NewMockClient(mockResponse, http.StatusOK)

		snapshot := Snapshot{Timestamp: "20220101000000", Original: "http://example.com/"}
		data, err := DownloadSnapshot(snapshot)
		if err != nil {
			t.Fatalf("DownloadSnapshot returned an unexpected error: %v", err)
		}
		if string(data) != mockResponse {
			t.Errorf("Expected response body '%s', got '%s'", mockResponse, string(data))
		}
	})
}

func TestRewriteLinks(t *testing.T) {
	baseURL, _ := url.Parse("http://example.com")
	htmlContent := `
		<html><body>
			<a href="https://web.archive.org/web/20220101000000/http://example.com/page1">Page 1</a>
			<a href="https://web.archive.org/web/20220101000000/http://othersite.com/page2">Page 2</a>
			<a href="/relative/path">Relative Path</a>
			<img src="https://web.archive.org/web/20220101000000/http://example.com/image.jpg" />
		</body></html>
	`
	rewritten, err := RewriteLinks([]byte(htmlContent), baseURL)
	if err != nil {
		t.Fatalf("RewriteLinks returned an unexpected error: %v", err)
	}

	if !strings.Contains(string(rewritten), `href="/page1"`) {
		t.Error("Expected link to be rewritten to /page1")
	}
	if !strings.Contains(string(rewritten), `href="https://web.archive.org/web/20220101000000/http://othersite.com/page2"`) {
		t.Error("External link should not have been rewritten")
	}
	if !strings.Contains(string(rewritten), `href="/relative/path"`) {
		t.Error("Relative link should not have been changed")
	}
	if !strings.Contains(string(rewritten), `src="/image.jpg"`) {
		t.Error("Expected image src to be rewritten to /image.jpg")
	}
}
