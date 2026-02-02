package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
)

type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	CommentsURL string `json:"comments_url"`
}

type Comment struct {
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

func (g *githubClient) GetIssues(ctx context.Context, owner, repo string) (*datanode.DataNode, error) {
	dn := datanode.New()
	client := NewAuthenticatedClient(ctx)
	apiURL := "https://api.github.com"
	if g.apiURL != "" {
		apiURL = g.apiURL
	}
	url := fmt.Sprintf("%s/repos/%s/%s/issues", apiURL, owner, repo)

	var allIssues []Issue
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Borg-Data-Collector")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch issues: %s", resp.Status)
		}

		var issues []Issue
		if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
			return nil, err
		}

		allIssues = append(allIssues, issues...)

		linkHeader := resp.Header.Get("Link")
		url = g.findNextURL(linkHeader)
	}

	for _, issue := range allIssues {
		var markdown strings.Builder
		markdown.WriteString(fmt.Sprintf("# Issue %d: %s\n\n", issue.Number, issue.Title))
		markdown.WriteString(fmt.Sprintf("**Author**: %s\n", issue.User.Login))
		markdown.WriteString(fmt.Sprintf("**State**: %s\n", issue.State))
		markdown.WriteString(fmt.Sprintf("**Created**: %s\n", issue.CreatedAt.Format(time.RFC1123)))
		markdown.WriteString(fmt.Sprintf("**Updated**: %s\n\n", issue.UpdatedAt.Format(time.RFC1123)))

		if len(issue.Labels) > 0 {
			markdown.WriteString("**Labels**:\n")
			for _, label := range issue.Labels {
				markdown.WriteString(fmt.Sprintf("- %s\n", label.Name))
			}
			markdown.WriteString("\n")
		}

		markdown.WriteString("## Body\n\n")
		markdown.WriteString(issue.Body)
		markdown.WriteString("\n\n")

		// Fetch comments
		comments, err := g.getComments(ctx, issue.CommentsURL)
		if err != nil {
			return nil, err
		}

		if len(comments) > 0 {
			markdown.WriteString("## Comments\n\n")
			for _, comment := range comments {
				markdown.WriteString(fmt.Sprintf("**%s** commented on %s:\n\n", comment.User.Login, comment.CreatedAt.Format(time.RFC1123)))
				markdown.WriteString(comment.Body)
				markdown.WriteString("\n\n---\n\n")
			}
		}

		filename := fmt.Sprintf("issues/%d.md", issue.Number)
		dn.AddData(filename, []byte(markdown.String()))
	}

	// Add an index file
	index, err := json.MarshalIndent(allIssues, "", "  ")
	if err != nil {
		return nil, err
	}
	dn.AddData("issues/INDEX.json", index)

	return dn, nil
}

func (g *githubClient) getComments(ctx context.Context, url string) ([]Comment, error) {
	client := NewAuthenticatedClient(ctx)
	var allComments []Comment

	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Borg-Data-Collector")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch comments: %s", resp.Status)
		}

		var comments []Comment
		if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)

		linkHeader := resp.Header.Get("Link")
		url = g.findNextURL(linkHeader)
	}

	return allComments, nil
}
