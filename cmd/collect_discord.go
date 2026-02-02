package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// collectDiscordCmd represents the collect discord command
var collectDiscordCmd = &cobra.Command{
	Use:   "discord",
	Short: "Collect a Discord server export.",
	Long:  `Collect a Discord server export from DiscordChatExporter and store it in a searchable archive.`,
}

// DiscordExport represents the top-level structure of a DiscordChatExporter JSON export.
// This struct is based on a common format, but may need adjustments for different export versions.
type DiscordExport struct {
	Guild    Guild     `json:"guild"`
	Channels []Channel `json:"channels"`
	Messages []Message `json:"messages"`
}

// Guild represents the server information.
type Guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Channel represents a channel in the server.
type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Message represents a single message in a channel.
type Message struct {
	ID        string       `json:"id"`
	ChannelID string       `json:"channelId"`
	Author    Author       `json:"author"`
	Timestamp time.Time    `json:"timestamp"`
	Content   string       `json:"content"`
	Attachments []Attachment `json:"attachments"`
}

// Author represents the message author.
type Author struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl"`
}

// Attachment represents a file attached to a message.
type Attachment struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
}

// sanitizeFilename removes characters that are invalid in file paths.
func sanitizeFilename(name string) string {
	// Replace path separators and other problematic characters with a dash.
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		}
		return r
	}, name)
}

var collectDiscordImportCmd = &cobra.Command{
	Use:   "import [path]",
	Short: "Import a DiscordChatExporter JSON export.",
	Long:  `Import a DiscordChatExporter JSON export and convert it to a searchable archive.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		fmt.Println("Importing Discord export from:", filePath)

		// Read the JSON file
		jsonFile, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("could not open file: %w", err)
		}
		defer jsonFile.Close()

		byteValue, err := io.ReadAll(jsonFile)
		if err != nil {
			return fmt.Errorf("could not read file: %w", err)
		}

		// Unmarshal the JSON data
		var export DiscordExport
		if err := json.Unmarshal(byteValue, &export); err != nil {
			return fmt.Errorf("could not unmarshal json: %w", err)
		}

		// Group messages by channel
		messagesByChannel := make(map[string][]Message)
		for _, msg := range export.Messages {
			messagesByChannel[msg.ChannelID] = append(messagesByChannel[msg.ChannelID], msg)
		}

		// Sanitize server name for the directory path
		sanitizedServerName := sanitizeFilename(export.Guild.Name)

		// Create a searchable index
		type SearchEntry struct {
			ID        string    `json:"id"`
			Channel   string    `json:"channel"`
			Author    string    `json:"author"`
			Timestamp time.Time `json:"timestamp"`
			Content   string    `json:"content"`
		}

		channelNames := make(map[string]string)
		for _, ch := range export.Channels {
			channelNames[ch.ID] = ch.Name
		}

		var searchIndex []SearchEntry
		for _, msg := range export.Messages {
			searchIndex = append(searchIndex, SearchEntry{
				ID:        msg.ID,
				Channel:   channelNames[msg.ChannelID],
				Author:    msg.Author.Name,
				Timestamp: msg.Timestamp,
				Content:   msg.Content,
			})
		}

		// Create the main output directory
		outputDir := filepath.Join("discord", sanitizedServerName)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("could not create output directory: %w", err)
		}

		// Save the index to a file
		indexData, err := json.MarshalIndent(searchIndex, "", "  ")
		if err != nil {
			return fmt.Errorf("could not marshal search index: %w", err)
		}

		indexPath := filepath.Join(outputDir, "INDEX.json")
		if err := os.WriteFile(indexPath, indexData, 0644); err != nil {
			return fmt.Errorf("could not write search index: %w", err)
		}

		// Process each channel and convert messages to Markdown
		for _, channel := range export.Channels {
			// Sort messages by timestamp
			sort.Slice(messagesByChannel[channel.ID], func(i, j int) bool {
				return messagesByChannel[channel.ID][i].Timestamp.Before(messagesByChannel[channel.ID][j].Timestamp)
			})

			var markdownContent strings.Builder
			markdownContent.WriteString(fmt.Sprintf("# %s\n\n", channel.Name))

			for _, msg := range messagesByChannel[channel.ID] {
				markdownContent.WriteString("---\n")
				markdownContent.WriteString(fmt.Sprintf("**%s** `%s`\n\n", msg.Author.Name, msg.Timestamp.Format("2006-01-02 15:04:05")))
				markdownContent.WriteString(msg.Content)
				markdownContent.WriteString("\n")

				for _, att := range msg.Attachments {
					// Download attachment
					resp, err := http.Get(att.URL)
					if err != nil {
						// Log the error but don't block the entire process
						fmt.Printf("Warning: could not download attachment %s: %v\n", att.URL, err)
						markdownContent.WriteString(fmt.Sprintf("\n[Failed to download %s](%s)", att.FileName, att.URL))
						continue
					}

					// Create attachments directory
					attachmentsDir := filepath.Join(outputDir, "attachments")
					if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
						return fmt.Errorf("could not create attachments directory: %w", err)
					}

					// Save attachment
					sanitizedAttachmentName := sanitizeFilename(att.FileName)
					attachmentPath := filepath.Join(attachmentsDir, sanitizedAttachmentName)
					outFile, err := os.Create(attachmentPath)
					if err != nil {
						resp.Body.Close()
						return fmt.Errorf("could not create attachment file: %w", err)
					}

					if _, err := io.Copy(outFile, resp.Body); err != nil {
						outFile.Close()
						resp.Body.Close()
						return fmt.Errorf("could not save attachment: %w", err)
					}
					outFile.Close()
					resp.Body.Close()

					// Update markdown to link to local file
					localPath := filepath.Join("..", "attachments", sanitizedAttachmentName)
					markdownContent.WriteString(fmt.Sprintf("\n[%s](%s)", att.FileName, localPath))
				}
				markdownContent.WriteString("\n\n")
			}

			// Create the output directory for markdown files
			channelsDir := filepath.Join(outputDir, "channels")
			if err := os.MkdirAll(channelsDir, 0755); err != nil {
				return fmt.Errorf("could not create output directory: %w", err)
			}

			// Sanitize channel name for the filename
			sanitizedChannelName := sanitizeFilename(channel.Name)

			// Write the markdown to a file
			filePath := filepath.Join(channelsDir, fmt.Sprintf("%s.md", sanitizedChannelName))
			if err := os.WriteFile(filePath, []byte(markdownContent.String()), 0644); err != nil {
				return fmt.Errorf("could not write markdown file for channel %s: %w", channel.Name, err)
			}
		}

		fmt.Printf("Successfully created archive in discord/%s\n", sanitizedServerName)
		return nil
	},
}

func init() {
	collectCmd.AddCommand(collectDiscordCmd)
	collectDiscordCmd.AddCommand(collectDiscordImportCmd)
}
