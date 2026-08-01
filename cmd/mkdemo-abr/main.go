// mkdemo-abr creates an ABR (Adaptive Bitrate) demo set from a source video.
// It uses ffmpeg to transcode to multiple bitrates, then encrypts each as v3 chunked SMSG.
//
// Usage: mkdemo-abr <input-video> <output-dir> [password]
//
// Output:
//
//	output-dir/manifest.json    - ABR manifest listing all variants
//	output-dir/track-1080p.smsg - 1080p variant (5 Mbps)
//	output-dir/track-720p.smsg  - 720p variant (2.5 Mbps)
//	output-dir/track-480p.smsg  - 480p variant (1 Mbps)
//	output-dir/track-360p.smsg  - 360p variant (500 Kbps)
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Snider/Borg/pkg/smsg"
)

// Preset defines a quality level for transcoding
type Preset struct {
	Name    string
	Width   int
	Height  int
	Bitrate string // For ffmpeg (e.g., "5M")
	BPS     int    // Bits per second for manifest
}

// Default presets matching ABRPresets in types.go
var presets = []Preset{
	{"1080p", 1920, 1080, "5M", 5000000},
	{"720p", 1280, 720, "2.5M", 2500000},
	{"480p", 854, 480, "1M", 1000000},
	{"360p", 640, 360, "500K", 500000},
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: mkdemo-abr <input-video> <output-dir> [password]")
		fmt.Println()
		fmt.Println("Creates ABR variant set from source video using ffmpeg.")
		fmt.Println()
		fmt.Println("Output:")
		fmt.Println("  output-dir/manifest.json    - ABR manifest")
		fmt.Println("  output-dir/track-1080p.smsg - 1080p (5 Mbps)")
		fmt.Println("  output-dir/track-720p.smsg  - 720p (2.5 Mbps)")
		fmt.Println("  output-dir/track-480p.smsg  - 480p (1 Mbps)")
		fmt.Println("  output-dir/track-360p.smsg  - 360p (500 Kbps)")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputDir := os.Args[2]

	// Check ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Println("Error: ffmpeg not found in PATH")
		fmt.Println("Install ffmpeg: https://ffmpeg.org/download.html")
		os.Exit(1)
	}

	// Generate or use provided password
	var password string
	if len(os.Args) > 3 {
		password = os.Args[3]
	} else {
		passwordBytes := make([]byte, 24)
		if _, err := rand.Read(passwordBytes); err != nil {
			fmt.Printf("Failed to generate password: %v\n", err)
			os.Exit(1)
		}
		password = base64.RawURLEncoding.EncodeToString(passwordBytes)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Get title from input filename
	title := filepath.Base(inputFile)
	ext := filepath.Ext(title)
	if ext != "" {
		title = title[:len(title)-len(ext)]
	}

	// Create ABR manifest
	manifest := smsg.NewABRManifest(title)

	fmt.Printf("Creating ABR variants for: %s\n", inputFile)
	fmt.Printf("Output directory: %s\n", outputDir)
	fmt.Printf("Password: %s\n\n", password)

	// Process each preset
	for _, preset := range presets {
		fmt.Printf("Processing %s (%dx%d @ %s)...\n", preset.Name, preset.Width, preset.Height, preset.Bitrate)

		// Step 1: Transcode with ffmpeg
		tempFile := filepath.Join(outputDir, fmt.Sprintf("temp-%s.mp4", preset.Name))
		if err := transcode(inputFile, tempFile, preset); err != nil {
			fmt.Printf("  Warning: Transcode failed for %s: %v\n", preset.Name, err)
			fmt.Printf("  Skipping this variant...\n")
			continue
		}

		// Step 2: Read transcoded file
		content, err := os.ReadFile(tempFile)
		if err != nil {
			fmt.Printf("  Error reading transcoded file: %v\n", err)
			os.Remove(tempFile)
			continue
		}

		// Step 3: Create SMSG message
		msg := smsg.NewMessage("dapp.fm ABR Demo")
		msg.Subject = fmt.Sprintf("%s - %s", title, preset.Name)
		msg.From = "dapp.fm"
		msg.AddBinaryAttachment(
			fmt.Sprintf("%s-%s.mp4", strings.ReplaceAll(title, " ", "_"), preset.Name),
			content,
			"video/mp4",
		)

		// Step 4: Create manifest for this variant
		variantManifest := smsg.NewManifest(title)
		variantManifest.LicenseType = "perpetual"
		variantManifest.Format = "dapp.fm/abr-v1"

		// Step 5: Encrypt with v3 chunked format
		params := &smsg.StreamParams{
			License:   password,
			ChunkSize: smsg.DefaultChunkSize, // 1MB chunks
		}

		encrypted, err := smsg.EncryptV3(msg, params, variantManifest)
		if err != nil {
			fmt.Printf("  Error encrypting: %v\n", err)
			os.Remove(tempFile)
			continue
		}

		// Step 6: Write SMSG file
		smsgFile := filepath.Join(outputDir, fmt.Sprintf("track-%s.smsg", preset.Name))
		if err := os.WriteFile(smsgFile, encrypted, 0644); err != nil {
			fmt.Printf("  Error writing SMSG: %v\n", err)
			os.Remove(tempFile)
			continue
		}

		// Step 7: Get chunk count from header
		header, err := smsg.GetV3Header(encrypted)
		if err != nil {
			fmt.Printf("  Warning: Could not read header: %v\n", err)
		}
		chunkCount := 0
		if header != nil && header.Chunked != nil {
			chunkCount = header.Chunked.TotalChunks
		}

		// Step 8: Add variant to manifest
		variant := smsg.Variant{
			Name:       preset.Name,
			Bandwidth:  preset.BPS,
			Width:      preset.Width,
			Height:     preset.Height,
			Codecs:     "avc1.640028,mp4a.40.2",
			URL:        fmt.Sprintf("track-%s.smsg", preset.Name),
			ChunkCount: chunkCount,
			FileSize:   int64(len(encrypted)),
		}
		manifest.AddVariant(variant)

		// Clean up temp file
		os.Remove(tempFile)

		fmt.Printf("  Created: %s (%d bytes, %d chunks)\n", smsgFile, len(encrypted), chunkCount)
	}

	if len(manifest.Variants) == 0 {
		fmt.Println("\nError: No variants created. Check ffmpeg output.")
		os.Exit(1)
	}

	// Write ABR manifest
	manifestPath := filepath.Join(outputDir, "manifest.json")
	if err := smsg.WriteABRManifest(manifest, manifestPath); err != nil {
		fmt.Printf("Failed to write manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Created ABR manifest: %s\n", manifestPath)
	fmt.Printf("✓ Variants: %d\n", len(manifest.Variants))
	fmt.Printf("✓ Default: %s\n", manifest.Variants[manifest.DefaultIdx].Name)
	fmt.Printf("\nMaster Password: %s\n", password)
	fmt.Println("\nStore this password securely - it decrypts ALL variants!")
}

// transcode uses ffmpeg to transcode the input to the specified preset
func transcode(input, output string, preset Preset) error {
	args := []string{
		"-i", input,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			preset.Width, preset.Height, preset.Width, preset.Height),
		"-c:v", "libx264",
		"-preset", "medium",
		"-b:v", preset.Bitrate,
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y", // Overwrite output
		output,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr // Show ffmpeg output for debugging

	return cmd.Run()
}
