// Example: Creating encrypted support reply messages
//
// This example demonstrates how to create password-protected secure messages
// that can be decrypted client-side using the BorgSMSG WASM module.
//
// Usage:
//
//	go run main.go
//	go run main.go -password "secret123" -body "Your message here"
//	go run main.go -password "secret123" -body "Message" -hint "Your hint"
//	go run main.go -password "secret123" -body "Message" -attach file.txt
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Snider/Borg/pkg/smsg"
	"github.com/Snider/Borg/pkg/stmf"
)

func main() {
	// Command line flags
	password := flag.String("password", "demo123", "Password for encryption")
	hint := flag.String("hint", "", "Optional password hint")
	body := flag.String("body", "", "Message body (if empty, uses demo content)")
	subject := flag.String("subject", "", "Message subject")
	from := flag.String("from", "support@example.com", "Sender address")
	attachFile := flag.String("attach", "", "File to attach (optional)")
	withReplyKey := flag.Bool("reply-key", false, "Include a reply public key")
	outputFile := flag.String("out", "", "Output file (if empty, prints to stdout)")
	rawBytes := flag.Bool("raw", false, "Output raw bytes instead of base64")

	flag.Parse()

	// Create the message
	var msg *smsg.Message
	if *body == "" {
		msg = createDemoMessage()
	} else {
		msg = smsg.NewMessage(*body)
	}

	// Set optional fields
	if *subject != "" {
		msg.WithSubject(*subject)
	}
	if *from != "" {
		msg.WithFrom(*from)
	}

	// Add attachment if specified
	if *attachFile != "" {
		if err := addAttachment(msg, *attachFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding attachment: %v\n", err)
			os.Exit(1)
		}
	}

	// Add reply key if requested
	if *withReplyKey {
		kp, err := stmf.GenerateKeyPair()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating reply key: %v\n", err)
			os.Exit(1)
		}
		msg.WithReplyKey(kp.PublicKeyBase64())
		fmt.Fprintf(os.Stderr, "Reply private key (keep secret): %s\n", kp.PrivateKeyBase64())
	}

	// Encrypt the message
	var encrypted []byte
	var err error
	if *hint != "" {
		encrypted, err = smsg.EncryptWithHint(msg, *password, *hint)
	} else {
		encrypted, err = smsg.Encrypt(msg, *password)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encryption failed: %v\n", err)
		os.Exit(1)
	}

	// Output the result
	var output []byte
	if *rawBytes {
		output = encrypted
	} else {
		output = []byte(base64.StdEncoding.EncodeToString(encrypted))
	}

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, output, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Encrypted message written to: %s\n", *outputFile)
	} else {
		fmt.Println(string(output))
	}

	// Print info
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "--- Message Info ---")
	fmt.Fprintf(os.Stderr, "Password: %s\n", *password)
	if *hint != "" {
		fmt.Fprintf(os.Stderr, "Hint: %s\n", *hint)
	}
	fmt.Fprintf(os.Stderr, "From: %s\n", msg.From)
	if msg.Subject != "" {
		fmt.Fprintf(os.Stderr, "Subject: %s\n", msg.Subject)
	}
	if len(msg.Attachments) > 0 {
		fmt.Fprintf(os.Stderr, "Attachments: %d\n", len(msg.Attachments))
	}
	if msg.ReplyKey != nil {
		fmt.Fprintln(os.Stderr, "Reply Key: included")
	}
}

// createDemoMessage creates a sample support reply message
func createDemoMessage() *smsg.Message {
	return smsg.NewMessage(`Hello,

Thank you for contacting our support team. We have reviewed your request and are pleased to provide the following update.

Your account has been verified and all services are now active. If you have any further questions, please don't hesitate to reach out.

Best regards,
The Support Team`).
		WithSubject("Re: Your Support Request #" + fmt.Sprintf("%d", time.Now().Unix()%100000)).
		WithFrom("support@example.com").
		SetMeta("ticket_id", fmt.Sprintf("%d", time.Now().Unix()%100000)).
		SetMeta("priority", "normal")
}

// addAttachment reads a file and adds it as an attachment
func addAttachment(msg *smsg.Message, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	name := filepath.Base(filePath)
	content := base64.StdEncoding.EncodeToString(data)
	mimeType := detectMimeType(filePath)

	msg.AddAttachment(name, content, mimeType)
	return nil
}

// detectMimeType returns a basic mime type based on file extension
func detectMimeType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}
