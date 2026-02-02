package bitcointalk

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

var (
	httpClient = &http.Client{}
	baseURL    = "https://bitcointalk.org"
)

// Post represents a single post in a thread.
type Post struct {
	Author  string
	Date    string
	Content string
}

// Thread represents a BitcoinTalk thread.
type Thread struct {
	Title string
	Posts []Post
}

// User represents a BitcoinTalk user profile.
type User struct {
	Username string
	// Add other user fields here as needed
}

// SearchResult represents a single result from a search.
type SearchResult struct {
	Title string
	URL   string
}

// ScrapeThread scrapes a full BitcoinTalk thread, handling pagination.
func ScrapeThread(threadID string) (*Thread, error) {
	var allPosts []Post
	var threadTitle string

	for page := 1; ; page++ {
		threadPage, err := ScrapeThreadPage(threadID, page)
		if err != nil {
			return nil, err
		}

		if page == 1 {
			threadTitle = threadPage.Title
		}

		if len(threadPage.Posts) == 0 {
			// No more posts, we're done
			break
		}

		allPosts = append(allPosts, threadPage.Posts...)

		// Be a good citizen and don't spam the server
		time.Sleep(1 * time.Second)
	}

	thread := &Thread{
		Title: threadTitle,
		Posts: allPosts,
	}

	return thread, nil
}

// ScrapeThreadPage scrapes a single page of a BitcoinTalk thread.
func ScrapeThreadPage(threadID string, page int) (*Thread, error) {
	// BitcoinTalk uses a 'topic' query parameter for the thread ID,
	// and the page number is determined by the '.<offset>' in the URL.
	// The offset is (page - 1) * 20, where 20 is the number of posts per page.
	offset := (page - 1) * 20
	url := fmt.Sprintf("%s/index.php?topic=%s.%d", baseURL, threadID, offset)

	// Make HTTP request
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	// Extract the thread title
	title := doc.Find("div.subject a").First().Text()
	title = strings.TrimSpace(title)
	if title == "" {
		// Fallback to the title tag if the subject is not found
		title = doc.Find("title").Text()
		title = strings.TrimSpace(title)
	}

	var posts []Post
	converter := md.NewConverter("", true, nil)

	// Find each post
	doc.Find("td.windowbg, td.windowbg2").Each(func(i int, s *goquery.Selection) {
		// This is a simplistic selector and might need refinement.
		// For now, we'll assume it correctly selects post containers.
		post := Post{}

		// Extract author
		author := s.Find("b a").First().Text()
		post.Author = strings.TrimSpace(author)

		// Extract date
		date := s.Find("td.td_headerandpost div.smalltext").First().Text()
		post.Date = strings.TrimSpace(date)

		// Extract content and convert to markdown
		contentHTML, err := s.Find("div.post").Html()
		if err == nil {
			markdown, err := converter.ConvertString(contentHTML)
			if err == nil {
				post.Content = markdown
			}
		}

		if post.Author != "" {
			posts = append(posts, post)
		}
	})

	thread := &Thread{
		Title: title,
		Posts: posts,
	}

	return thread, nil
}

// ScrapeUserPage scrapes a BitcoinTalk user profile.
func ScrapeUserPage(userID string) (*User, error) {
	// Be a good citizen and don't spam the server
	time.Sleep(1 * time.Second)

	url := fmt.Sprintf("%s/index.php?action=profile;u=%s", baseURL, userID)

	// Make HTTP request
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	// Extract the username
	username := doc.Find("td.windowbg td:contains('Name:')").Next().Text()
	username = strings.TrimSpace(username)

	user := &User{
		Username: username,
	}
	if user.Username == "" {
		return nil, fmt.Errorf("could not find username")
	}

	return user, nil
}

// ScrapeSearchPage scrapes a BitcoinTalk search results page.
func ScrapeSearchPage(query string) ([]SearchResult, error) {
	// Be a good citizen and don't spam the server
	time.Sleep(1 * time.Second)

	searchURL := fmt.Sprintf("%s/index.php?action=search2;search=%s", baseURL, query)

	// Make HTTP request
	res, err := httpClient.Get(searchURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	// Find each result
	doc.Find("td.windowbg2 a").Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		link, exists := s.Attr("href")
		if exists {
			results = append(results, SearchResult{
				Title: title,
				URL:   link,
			})
		}
	})

	return results, nil
}

// SetBaseURL sets the base URL for the bitcointalk scraper.
// This is useful for testing.
func SetBaseURL(url string) {
	baseURL = url
}
