package bitcointalk

import (
	"bytes"
	_ "embed"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/Snider/Borg/pkg/mocks"
)

//go:embed thread_6.html
var threadHTML []byte

//go:embed user_3.html
var userHTML []byte

func TestScrapeThreadPage(t *testing.T) {
	// Create a mock HTTP client that returns the test data
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://bitcointalk.org/index.php?topic=6.0": {
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(threadHTML)),
		},
	})
	httpClient = mockClient

	// Call the function being tested
	thread, err := ScrapeThreadPage("6", 1)
	if err != nil {
		t.Fatalf("error scraping thread page: %v", err)
	}

	// Check the results
	if thread.Title != "Repost: Bitcoin Maturation" {
		t.Errorf("unexpected title: got %q, want %q", thread.Title, "Repost: Bitcoin Maturation")
	}

	if len(thread.Posts) != 7 {
		t.Errorf("unexpected number of posts: got %d, want %d", len(thread.Posts), 7)
	}

	if thread.Posts[0].Author != "satoshi" {
		t.Errorf("unexpected author: got %q, want %q", thread.Posts[0].Author, "satoshi")
	}
}

func TestScrapeUserPage(t *testing.T) {
	// Create a mock HTTP client that returns the test data
	mockClient := mocks.NewMockClient(map[string]*http.Response{
		"https://bitcointalk.org/index.php?action=profile;u=3": {
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(userHTML)),
		},
	})
	httpClient = mockClient

	// Call the function being tested
	user, err := ScrapeUserPage("3")
	if err != nil {
		t.Fatalf("error scraping user page: %v", err)
	}

	// Check the results
	if user.Username != "satoshi" {
		t.Errorf("unexpected username: got %q, want %q", user.Username, "satoshi")
	}
}

func TestScrapeSearchPage(t *testing.T) {
	// This test requires a search results page, which I haven't downloaded yet.
	// I'll skip this test for now and come back to it later.
	t.Skip("Skipping test: requires search results page")
}
