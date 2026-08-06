package telegram

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Snider/Borg/pkg/datanode"
)

// TelegramExport represents the overall structure of the Telegram JSON export.
type TelegramExport struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	ID       int       `json:"id"`
	Messages []Message `json:"messages"`
}

// Message represents a single message in the Telegram export.
type Message struct {
	ID        int           `json:"id"`
	Type      string        `json:"type"`
	Date      string        `json:"date"`
	From      string        `json:"from"`
	Text      interface{}   `json:"text"`
	File      string        `json:"file"`
	ReplyToID int           `json:"reply_to_message_id"`
	Photo     string        `json:"photo"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	ForwardedFrom string    `json:"forwarded_from"`
}

// TextEntity represents a formatted part of a message text.
type TextEntity struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Href string `json:"href,omitempty"`
}

// parseText converts the 'text' field (which can be a string or a slice of entities)
// into a Markdown formatted string.
func parseText(text interface{}) string {
	switch v := text.(type) {
	case string:
		return v
	case []interface{}:
		var builder strings.Builder
		for _, item := range v {
			switch e := item.(type) {
			case string:
				builder.WriteString(e)
			case map[string]interface{}:
				// A simple approach to convert map to TextEntity
				var entity TextEntity
				if t, ok := e["type"].(string); ok {
					entity.Type = t
				}
				if t, ok := e["text"].(string); ok {
					entity.Text = t
				}
				if h, ok := e["href"].(string); ok {
					entity.Href = h
				}

				switch entity.Type {
				case "bold":
					builder.WriteString(fmt.Sprintf("**%s**", entity.Text))
				case "italic":
					builder.WriteString(fmt.Sprintf("*%s*", entity.Text))
				case "link", "text_link":
					builder.WriteString(fmt.Sprintf("[%s](%s)", entity.Text, entity.Href))
				case "pre", "code":
					builder.WriteString(fmt.Sprintf("`%s`", entity.Text))
				default:
					builder.WriteString(entity.Text)
				}
			}
		}
		return builder.String()
	}
	return ""
}

// Parse parses a Telegram export directory and returns a DataNode.
func Parse(path string) (*datanode.DataNode, error) {
	jsonPath := filepath.Join(path, "result.json")
	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read result.json: %w", err)
	}

	var export TelegramExport
	if err := json.Unmarshal(jsonBytes, &export); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	dn := datanode.New()
	channelName := export.Name

	// Create INDEX.json
	indexData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal index data: %w", err)
	}
	indexPath := filepath.Join("telegram", channelName, "INDEX.json")
	dn.AddData(indexPath, indexData)

	messagesByMonth := make(map[string][]Message)
	for _, msg := range export.Messages {
		if msg.Type != "message" {
			continue
		}
		t, err := time.Parse("2006-01-02T15:04:05", msg.Date)
		if err != nil {
			continue // Skip messages with invalid date format
		}
		month := t.Format("2006-01")
		messagesByMonth[month] = append(messagesByMonth[month], msg)
	}

	for month, messages := range messagesByMonth {
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Date < messages[j].Date
		})

		var mdBuilder strings.Builder
		for _, msg := range messages {
			mdBuilder.WriteString(fmt.Sprintf("### %s from %s (ID: %d)\n\n", msg.Date, msg.From, msg.ID))
			if msg.ReplyToID != 0 {
				mdBuilder.WriteString(fmt.Sprintf("> Reply to message %d\n\n", msg.ReplyToID))
			}
			if msg.ForwardedFrom != "" {
				mdBuilder.WriteString(fmt.Sprintf("> Forwarded from %s\n\n", msg.ForwardedFrom))
			}

			text := parseText(msg.Text)
			mdBuilder.WriteString(text)
			mdBuilder.WriteString("\n\n")

			mediaPath := ""
			if msg.File != "" {
				mediaPath = msg.File
			} else if msg.Photo != "" {
				mediaPath = msg.Photo
			}

			if mediaPath != "" {
				mdBuilder.WriteString(fmt.Sprintf("![Media](media/%s)\n\n", filepath.Base(mediaPath)))

				srcMediaPath := filepath.Join(path, mediaPath)
				mediaBytes, err := os.ReadFile(srcMediaPath)
				if err == nil {
					destMediaPath := filepath.Join("telegram", channelName, "media", filepath.Base(mediaPath))
					dn.AddData(destMediaPath, mediaBytes)
				}
			}
			mdBuilder.WriteString("---\n\n")
		}

		mdPath := filepath.Join("telegram", channelName, "messages", month+".md")
		dn.AddData(mdPath, []byte(mdBuilder.String()))
	}

	return dn, nil
}
