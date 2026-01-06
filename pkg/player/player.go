// Package player provides the core media player functionality for dapp.fm
// It can be used both as Wails bindings (memory speed) or HTTP server (fallback)
package player

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Snider/Borg/pkg/smsg"
)

// Player provides media decryption and playback services
// Methods are exposed to JavaScript via Wails bindings
type Player struct {
	ctx context.Context
}

// NewPlayer creates a new Player instance
func NewPlayer() *Player {
	return &Player{}
}

// Startup is called when the Wails app starts
func (p *Player) Startup(ctx context.Context) {
	p.ctx = ctx
}

// DecryptResult holds the decrypted message data
type DecryptResult struct {
	Body        string           `json:"body"`
	Subject     string           `json:"subject,omitempty"`
	From        string           `json:"from,omitempty"`
	Attachments []AttachmentInfo `json:"attachments,omitempty"`
}

// AttachmentInfo describes a decrypted attachment
type AttachmentInfo struct {
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
	DataURL  string `json:"data_url"` // Base64 data URL for direct playback
}

// ManifestInfo holds public metadata (readable without decryption)
type ManifestInfo struct {
	Title         string      `json:"title,omitempty"`
	Artist        string      `json:"artist,omitempty"`
	Album         string      `json:"album,omitempty"`
	Genre         string      `json:"genre,omitempty"`
	Year          int         `json:"year,omitempty"`
	ReleaseType   string      `json:"release_type,omitempty"`
	Duration      int         `json:"duration,omitempty"`
	Format        string      `json:"format,omitempty"`
	ExpiresAt     int64       `json:"expires_at,omitempty"`
	IssuedAt      int64       `json:"issued_at,omitempty"`
	LicenseType   string      `json:"license_type,omitempty"`
	Tracks        []TrackInfo `json:"tracks,omitempty"`
	IsExpired     bool        `json:"is_expired"`
	TimeRemaining string      `json:"time_remaining,omitempty"`
}

// TrackInfo describes a track marker
type TrackInfo struct {
	Title    string  `json:"title"`
	Start    float64 `json:"start"`
	End      float64 `json:"end,omitempty"`
	Type     string  `json:"type,omitempty"`
	TrackNum int     `json:"track_num,omitempty"`
}

// GetManifest returns public metadata without decryption
// This is memory-speed via Wails bindings
func (p *Player) GetManifest(encrypted string) (*ManifestInfo, error) {
	info, err := smsg.GetInfoBase64(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	result := &ManifestInfo{}

	if info.Manifest != nil {
		m := info.Manifest
		result.Title = m.Title
		result.Artist = m.Artist
		result.Album = m.Album
		result.Genre = m.Genre
		result.Year = m.Year
		result.ReleaseType = m.ReleaseType
		result.Duration = m.Duration
		result.Format = m.Format
		result.ExpiresAt = m.ExpiresAt
		result.IssuedAt = m.IssuedAt
		result.LicenseType = m.LicenseType
		result.IsExpired = m.IsExpired()

		if !result.IsExpired && m.ExpiresAt > 0 {
			remaining := m.TimeRemaining()
			result.TimeRemaining = formatDurationSeconds(remaining)
		}

		for _, t := range m.Tracks {
			result.Tracks = append(result.Tracks, TrackInfo{
				Title:    t.Title,
				Start:    t.Start,
				End:      t.End,
				Type:     t.Type,
				TrackNum: t.TrackNum,
			})
		}
	}

	return result, nil
}

// IsLicenseValid checks if the license has expired
// This is memory-speed via Wails bindings
func (p *Player) IsLicenseValid(encrypted string) (bool, error) {
	info, err := smsg.GetInfoBase64(encrypted)
	if err != nil {
		return false, fmt.Errorf("failed to check license: %w", err)
	}

	if info.Manifest != nil && info.Manifest.ExpiresAt > 0 {
		return !info.Manifest.IsExpired(), nil
	}

	// No expiration set = perpetual license
	return true, nil
}

// Decrypt decrypts the SMSG content and returns playable media
// This is memory-speed via Wails bindings - no HTTP, no WASM
func (p *Player) Decrypt(encrypted string, password string) (*DecryptResult, error) {
	// Check license first
	valid, err := p.IsLicenseValid(encrypted)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("license has expired")
	}

	// Decrypt using pkg/smsg (Base64 variant for string input)
	msg, err := smsg.DecryptBase64(encrypted, password)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	result := &DecryptResult{
		Body:    msg.Body,
		Subject: msg.Subject,
		From:    msg.From,
	}

	// Convert attachments to data URLs for direct playback
	for _, att := range msg.Attachments {
		// Decode base64 content to get size
		data, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			continue
		}

		// Create data URL for the browser to play directly
		dataURL := fmt.Sprintf("data:%s;base64,%s", att.MimeType, att.Content)

		result.Attachments = append(result.Attachments, AttachmentInfo{
			Name:     att.Name,
			MimeType: att.MimeType,
			Size:     len(data),
			DataURL:  dataURL,
		})
	}

	return result, nil
}

// QuickDecrypt returns just the first attachment as a data URL
// Optimized for single-track playback
func (p *Player) QuickDecrypt(encrypted string, password string) (string, error) {
	result, err := p.Decrypt(encrypted, password)
	if err != nil {
		return "", err
	}

	if len(result.Attachments) == 0 {
		return "", fmt.Errorf("no media attachments found")
	}

	return result.Attachments[0].DataURL, nil
}

// GetLicenseInfo returns detailed license information
func (p *Player) GetLicenseInfo(encrypted string) (map[string]interface{}, error) {
	manifest, err := p.GetManifest(encrypted)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"is_valid":       !manifest.IsExpired,
		"license_type":   manifest.LicenseType,
		"time_remaining": manifest.TimeRemaining,
	}

	if manifest.ExpiresAt > 0 {
		info["expires_at"] = time.Unix(manifest.ExpiresAt, 0).Format(time.RFC3339)
	}
	if manifest.IssuedAt > 0 {
		info["issued_at"] = time.Unix(manifest.IssuedAt, 0).Format(time.RFC3339)
	}

	return info, nil
}

// Serve starts an HTTP server for CLI/fallback mode
// This is the slower TCP path - use Wails bindings when possible
func (p *Player) Serve(addr string) error {
	mux := http.NewServeMux()

	// Serve embedded assets
	mux.Handle("/", http.FileServer(http.FS(Assets)))

	// API endpoints for WASM fallback
	mux.HandleFunc("/api/manifest", p.handleManifest)
	mux.HandleFunc("/api/decrypt", p.handleDecrypt)
	mux.HandleFunc("/api/license", p.handleLicense)

	fmt.Printf("dapp.fm player serving at http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (p *Player) handleManifest(w http.ResponseWriter, r *http.Request) {
	encrypted := r.URL.Query().Get("data")
	if encrypted == "" {
		http.Error(w, "missing data parameter", http.StatusBadRequest)
		return
	}

	manifest, err := p.GetManifest(encrypted)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

func (p *Player) handleDecrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Encrypted string `json:"encrypted"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := p.Decrypt(req.Encrypted, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (p *Player) handleLicense(w http.ResponseWriter, r *http.Request) {
	encrypted := r.URL.Query().Get("data")
	if encrypted == "" {
		http.Error(w, "missing data parameter", http.StatusBadRequest)
		return
	}

	info, err := p.GetLicenseInfo(encrypted)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatDurationSeconds(seconds int64) string {
	if seconds < 0 {
		return "expired"
	}

	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
