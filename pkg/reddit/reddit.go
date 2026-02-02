package reddit

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Comment represents a single Reddit comment.
type Comment struct {
	Author string
	Body   string
}

// Thread represents a Reddit thread, including the original post and all comments.
type Thread struct {
	Title    string
	Post     string
	Comments []Comment
	URL      string
}

// ScrapeThread fetches and parses a Reddit thread from a given URL.
func ScrapeThread(url string) (*Thread, error) {
	// Make sure we're using old.reddit.com for simpler scraping
	if !strings.Contains(url, "old.reddit.com") {
		url = strings.Replace(url, "reddit.com", "old.reddit.com", 1)
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status: %s", res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	thread := &Thread{}

	// Scrape the post title and content
	thread.Title = doc.Find("a.title").First().Text()
	thread.Post = doc.Find("div.expando .md").First().Text()

	// Scrape comments
	doc.Find(".commentarea .comment").Each(func(i int, s *goquery.Selection) {
		author := s.Find(".author").First().Text()
		body := s.Find(".md").First().Text()
		thread.Comments = append(thread.Comments, Comment{Author: author, Body: body})
	})

	return thread, nil
}

// ScrapeSubreddit fetches and parses a subreddit's posts.
func ScrapeSubreddit(name, sort string, limit int) ([]*Thread, error) {
	url := fmt.Sprintf("https://old.reddit.com/r/%s/", name)
	if sort == "top" {
		url = fmt.Sprintf("https://old.reddit.com/r/%s/top/?t=all", name)
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status: %s", res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var threads []*Thread
	doc.Find("div.thing.link").Each(func(i int, s *goquery.Selection) {
		if i >= limit {
			return
		}
		title := s.Find("a.title").Text()
		postURL, _ := s.Find("a.title").Attr("href")
		if !strings.HasPrefix(postURL, "http") {
			postURL = "https://old.reddit.com" + postURL
		}
		threads = append(threads, &Thread{Title: title, URL: postURL})
	})

	return threads, nil
}

// ScrapeUser fetches and parses a user's posts.
func ScrapeUser(name string) ([]*Thread, error) {
	url := fmt.Sprintf("https://old.reddit.com/user/%s/", name)

	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status: %s", res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var threads []*Thread
	doc.Find("div.thing.link").Each(func(i int, s *goquery.Selection) {
		title := s.Find("a.title").Text()
		postURL, _ := s.Find("a.title").Attr("href")
		if !strings.HasPrefix(postURL, "http") {
			postURL = "https://old.reddit.com" + postURL
		}
		threads = append(threads, &Thread{Title: title, URL: postURL})
	})

	return threads, nil
}
