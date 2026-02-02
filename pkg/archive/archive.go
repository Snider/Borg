package archive

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

var BaseURL = "https://archive.org"

type Item struct {
	Identifier string `json:"identifier"`
}

type SearchResponse struct {
	Response struct {
		Docs []Item `json:"docs"`
	} `json:"response"`
}

type ItemMetadata struct {
	Files []File `json:"files"`
}

type File struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Format string `json:"format"`
	Size   string `json:"size"`
}

func Search(query, mediaType string, limit int) ([]Item, error) {
	var allItems []Item
	page := 1
	const rowsPerPage = 100 // A reasonable number of results per page

	for {
		baseURL, err := url.Parse(BaseURL + "/advancedsearch.php")
		if err != nil {
			return nil, err
		}

		params := url.Values{}
		params.Add("q", query)
		if mediaType != "" {
			params.Add("fq", "mediatype:"+mediaType)
		}
		params.Add("fl[]", "identifier")
		params.Add("output", "json")
		params.Add("page", fmt.Sprintf("%d", page))

		if limit == -1 {
			params.Add("rows", fmt.Sprintf("%d", rowsPerPage))
		} else {
			params.Add("rows", fmt.Sprintf("%d", limit))
		}

		baseURL.RawQuery = params.Encode()

		resp, err := http.Get(baseURL.String())
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("bad status: %s", resp.Status)
		}

		var searchResponse SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(searchResponse.Response.Docs) == 0 {
			break // No more results
		}

		allItems = append(allItems, searchResponse.Response.Docs...)

		if limit != -1 && len(allItems) >= limit {
			return allItems[:limit], nil
		}

		if limit != -1 {
			break // We only needed one page
		}

		page++
	}

	return allItems, nil
}

func GetItem(identifier string) (*ItemMetadata, error) {
	url := fmt.Sprintf("%s/metadata/%s", BaseURL, identifier)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var itemMetadata ItemMetadata
	if err := json.NewDecoder(resp.Body).Decode(&itemMetadata); err != nil {
		return nil, err
	}

	return &itemMetadata, nil
}

func DownloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func GetCollection(identifier string) ([]Item, error) {
	return Search(fmt.Sprintf("collection:%s", identifier), "", -1) // -1 for no limit
}

func DownloadItem(identifier, baseDir string, formatFilter string) error {
	item, err := GetItem(identifier)
	if err != nil {
		return fmt.Errorf("could not get item metadata for %s: %w", identifier, err)
	}

	itemDir := fmt.Sprintf("%s/%s", baseDir, identifier)
	if err := os.MkdirAll(itemDir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", itemDir, err)
	}

	// Save metadata
	metadataJSON, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal metadata for %s: %w", identifier, err)
	}
	if err := os.WriteFile(fmt.Sprintf("%s/metadata.json", itemDir), metadataJSON, 0644); err != nil {
		return fmt.Errorf("could not write metadata.json for %s: %w", identifier, err)
	}

	// Save file list
	filesJSON, err := json.MarshalIndent(item.Files, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal file list for %s: %w", identifier, err)
	}
	if err := os.WriteFile(fmt.Sprintf("%s/_files.json", itemDir), filesJSON, 0644); err != nil {
		return fmt.Errorf("could not write _files.json for %s: %w", identifier, err)
	}

	fmt.Printf("Downloading item %s...\n", identifier)
	for _, file := range item.Files {
		if formatFilter != "" && file.Format != formatFilter {
			continue
		}
		downloadURL := fmt.Sprintf("%s/download/%s/%s", BaseURL, identifier, file.Name)
		filePath := fmt.Sprintf("%s/%s", itemDir, file.Name)
		fmt.Printf("  Downloading file %s...\n", file.Name)
		if err := DownloadFile(downloadURL, filePath); err != nil {
			// Log error but continue trying other files
			fmt.Printf("    Error downloading %s: %v\n", file.Name, err)
		}
	}
	return nil
}
