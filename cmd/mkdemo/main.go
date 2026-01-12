// mkdemo creates an RFC-quality demo SMSG file with a cryptographically secure password
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/smsg"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: mkdemo <input-media-file> <output-smsg-file>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Read input file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Failed to read input file: %v\n", err)
		os.Exit(1)
	}

	// Use existing password or generate new one
	var password string
	if len(os.Args) > 3 {
		password = os.Args[3]
	} else {
		// Generate cryptographically secure password (32 bytes = 256 bits)
		passwordBytes := make([]byte, 24)
		if _, err := rand.Read(passwordBytes); err != nil {
			fmt.Printf("Failed to generate password: %v\n", err)
			os.Exit(1)
		}
		// Use base64url encoding, trimmed to 32 chars for readability
		password = base64.RawURLEncoding.EncodeToString(passwordBytes)
	}

	// Create manifest with filename as title
	title := filepath.Base(inputFile)
	ext := filepath.Ext(title)
	if ext != "" {
		title = title[:len(title)-len(ext)]
	}
	manifest := smsg.NewManifest(title)
	manifest.LicenseType = "perpetual"
	manifest.Format = "dapp.fm/v1"

	// Create message with attachment (using binary attachment for v2 format)
	msg := smsg.NewMessage("Welcome to dapp.fm - Zero-Trust DRM for the open web.")
	msg.Subject = "dapp.fm Demo"
	msg.From = "dapp.fm"
	msg.AddBinaryAttachment(
		filepath.Base(inputFile),
		content,
		"video/mp4",
	)

	// Encrypt with v2 binary format (smaller file size)
	encrypted, err := smsg.EncryptV2WithManifest(msg, password, manifest)
	if err != nil {
		fmt.Printf("Failed to encrypt: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := os.WriteFile(outputFile, encrypted, 0644); err != nil {
		fmt.Printf("Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created: %s (%d bytes)\n", outputFile, len(encrypted))
	fmt.Printf("Master Password: %s\n", password)
	fmt.Println("\nStore this password securely - it cannot be recovered!")
}
