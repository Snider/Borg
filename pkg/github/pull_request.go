package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
)

type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	MergedAt  time.Time `json:"merged_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Links struct {
		Comments struct {
			Href string `json:"href"`
		} `json:"comments"`
		ReviewComments struct {
			Href string `json:"href"`
		} `json:"review_comments"`
	} `json:"_links"`
	DiffURL string `json:"diff_url"`
}

type ReviewComment struct {
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

func (g *githubClient) GetPullRequests(ctx context.Context, owner, repo string) (*datanode.DataNode, error) {
	dn := datanode.New()
	client := NewAuthenticatedClient(ctx)
	apiURL := "https://api.github.com"
	if g.apiURL != "" {
		apiURL = g.apiURL
	}
	// Get both open and closed pull requests
	url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=all", apiURL, owner, repo)

	var allPRs []PullRequest
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
			return nil, fmt.Errorf("failed to fetch pull requests: %s", resp.Status)
		}

		var prs []PullRequest
		if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
			return nil, err
		}

		allPRs = append(allPRs, prs...)

		linkHeader := resp.Header.Get("Link")
		url = g.findNextURL(linkHeader)
	}

	for _, pr := range allPRs {
		var markdown strings.Builder
		markdown.WriteString(fmt.Sprintf("# PR %d: %s\n\n", pr.Number, pr.Title))
		markdown.WriteString(fmt.Sprintf("**Author**: %s\n", pr.User.Login))
		markdown.WriteString(fmt.Sprintf("**State**: %s\n", pr.State))
		markdown.WriteString(fmt.Sprintf("**Created**: %s\n", pr.CreatedAt.Format(time.RFC1123)))
		markdown.WriteString(fmt.Sprintf("**Updated**: %s\n", pr.UpdatedAt.Format(time.RFC1123)))
		if !pr.MergedAt.IsZero() {
			markdown.WriteString(fmt.Sprintf("**Merged**: %s\n", pr.MergedAt.Format(time.RFC1123)))
		}
		markdown.WriteString(fmt.Sprintf("\n**[View Diff](%s)**\n\n", pr.DiffURL))

		if len(pr.Labels) > 0 {
			markdown.WriteString("**Labels**:\n")
			for _, label := range pr.Labels {
				markdown.WriteString(fmt.Sprintf("- %s\n", label.Name))
			}
			markdown.WriteString("\n")
		}

		markdown.WriteString("## Body\n\n")
		markdown.WriteString(pr.Body)
		markdown.WriteString("\n\n")

		// Fetch diff
		diff, err := g.getDiff(ctx, pr.DiffURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get diff for PR #%d: %w", pr.Number, err)
		}
		dn.AddData(fmt.Sprintf("pulls/%d.diff", pr.Number), diff)

		// Fetch review comments
		reviewComments, err := g.getReviewComments(ctx, pr.Links.ReviewComments.Href)
		if err != nil {
			return nil, err
		}

		if len(reviewComments) > 0 {
			markdown.WriteString("## Review Comments\n\n")
			for _, comment := range reviewComments {
				markdown.WriteString(fmt.Sprintf("**%s** commented on `%s` at %s:\n\n", comment.User.Login, comment.Path, comment.CreatedAt.Format(time.RFC1123)))
				markdown.WriteString(comment.Body)
				markdown.WriteString("\n\n---\n\n")
			}
		}

		filename := fmt.Sprintf("pulls/%d.md", pr.Number)
		dn.AddData(filename, []byte(markdown.String()))
	}

	// Add an index file
	index, err := json.MarshalIndent(allPRs, "", "  ")
	if err != nil {
		return nil, err
	}
	dn.AddData("pulls/INDEX.json", index)

	return dn, nil
}

func (g *githubClient) getReviewComments(ctx context.Context, url string) ([]ReviewComment, error) {
	client := NewAuthenticatedClient(ctx)
	var allComments []ReviewComment

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
			return nil, fmt.Errorf("failed to fetch review comments: %s", resp.Status)
		}

		var comments []ReviewComment
		if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)

		linkHeader := resp.Header.Get("Link")
		url = g.findNextURL(linkHeader)
	}

	return allComments, nil
}

func (g *githubClient) getDiff(ctx context.Context, url string) ([]byte, error) {
	client := NewAuthenticatedClient(ctx)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Borg-Data-Collector")
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch diff: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
