// Package main demonstrates encrypting media files into SMSG format for dapp.fm
//
// Usage:
//
//	go run main.go -input video.mp4 -output video.smsg -password "license-token" -title "My Track" -artist "Artist Name"
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/smsg"
)

func main() {
	inputFile := flag.String("input", "", "Input media file (mp4, mp3, etc)")
	outputFile := flag.String("output", "", "Output SMSG file (default: input.smsg)")
	password := flag.String("password", "", "License token / password for encryption")
	title := flag.String("title", "", "Track title (default: filename)")
	artist := flag.String("artist", "", "Artist name")
	hint := flag.String("hint", "", "Optional password hint")
	outputBase64 := flag.Bool("base64", false, "Output as base64 text file instead of binary")

	flag.Parse()

	if *inputFile == "" {
		log.Fatal("Input file is required. Use -input flag.")
	}

	if *password == "" {
		log.Fatal("Password/license token is required. Use -password flag.")
	}

	// Read input file
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	// Determine MIME type
	ext := strings.ToLower(filepath.Ext(*inputFile))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Fallback for common types
		switch ext {
		case ".mp4":
			mimeType = "video/mp4"
		case ".mp3":
			mimeType = "audio/mpeg"
		case ".wav":
			mimeType = "audio/wav"
		case ".ogg":
			mimeType = "audio/ogg"
		case ".webm":
			mimeType = "video/webm"
		case ".m4a":
			mimeType = "audio/mp4"
		case ".flac":
			mimeType = "audio/flac"
		default:
			mimeType = "application/octet-stream"
		}
	}

	// Set defaults
	trackTitle := *title
	if trackTitle == "" {
		trackTitle = strings.TrimSuffix(filepath.Base(*inputFile), ext)
	}

	output := *outputFile
	if output == "" {
		output = *inputFile + ".smsg"
		if *outputBase64 {
			output = *inputFile + ".smsg.txt"
		}
	}

	// Create SMSG message with media attachment
	msg := smsg.NewMessage("Licensed media content from dapp.fm")
	msg.WithSubject(trackTitle)

	if *artist != "" {
		msg.WithFrom(*artist)
	}

	// Add the media file as base64 attachment
	contentB64 := base64.StdEncoding.EncodeToString(data)
	msg.AddAttachment(filepath.Base(*inputFile), contentB64, mimeType)

	// Encrypt
	var encrypted []byte
	if *hint != "" {
		encrypted, err = smsg.EncryptWithHint(msg, *password, *hint)
	} else {
		encrypted, err = smsg.Encrypt(msg, *password)
	}
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}

	// Write output
	if *outputBase64 {
		// Write as base64 text
		b64 := base64.StdEncoding.EncodeToString(encrypted)
		err = os.WriteFile(output, []byte(b64), 0644)
	} else {
		// Write as binary
		err = os.WriteFile(output, encrypted, 0644)
	}

	if err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("Encrypted media created successfully!\n")
	fmt.Printf("  Input:    %s (%s)\n", *inputFile, mimeType)
	fmt.Printf("  Output:   %s\n", output)
	fmt.Printf("  Title:    %s\n", trackTitle)
	if *artist != "" {
		fmt.Printf("  Artist:   %s\n", *artist)
	}
	fmt.Printf("  Size:     %.2f MB -> %.2f MB\n",
		float64(len(data))/1024/1024,
		float64(len(encrypted))/1024/1024)
	fmt.Printf("\nLicense token: %s\n", *password)
	fmt.Printf("\nShare the .smsg file publicly. Only users with the license token can play it.\n")
}
