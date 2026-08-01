// extract-demo extracts the video from a v2 SMSG file
package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/smsg"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: extract-demo <input.smsg> <password> <output.mp4>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	password := os.Args[2]
	outputFile := os.Args[3]

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Failed to read: %v\n", err)
		os.Exit(1)
	}

	// Get info first
	info, err := smsg.GetInfo(data)
	if err != nil {
		fmt.Printf("Failed to get info: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Format: %s, Compression: %s\n", info.Format, info.Compression)

	// Decrypt
	msg, err := smsg.Decrypt(data, password)
	if err != nil {
		fmt.Printf("Failed to decrypt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Body: %s...\n", msg.Body[:min(50, len(msg.Body))])
	fmt.Printf("Attachments: %d\n", len(msg.Attachments))

	if len(msg.Attachments) > 0 {
		att := msg.Attachments[0]
		fmt.Printf("  Name: %s, MIME: %s, Size: %d\n", att.Name, att.MimeType, att.Size)

		// Decode and save
		decoded, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			fmt.Printf("Failed to decode: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(outputFile, decoded, 0644); err != nil {
			fmt.Printf("Failed to save: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Saved to %s (%d bytes)\n", outputFile, len(decoded))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
