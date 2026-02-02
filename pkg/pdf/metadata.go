package pdf

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

// Metadata holds the extracted PDF metadata.
type Metadata struct {
	File     string   `json:"file"`
	Title    string   `json:"title"`
	Authors  []string `json:"authors"`
	Abstract string   `json:"abstract"`
	Pages    int      `json:"pages"`
	Created  string   `json:"created"`
}

// ExtractMetadata extracts metadata from a PDF file using the pdfinfo command.
func ExtractMetadata(filePath string) (*Metadata, error) {
	cmd := exec.Command("pdfinfo", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{File: filePath}
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Title":
			metadata.Title = value
		case "Author":
			metadata.Authors = strings.Split(value, ",")
		case "CreationDate":
			metadata.Created = value
		case "Pages":
			pages, err := strconv.Atoi(value)
			if err == nil {
				metadata.Pages = pages
			}
		}
	}

	return metadata, nil
}
