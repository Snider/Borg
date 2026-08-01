// dapp-fm-app is a native desktop media player for dapp.fm
// Decryption in Go, media served via Wails asset handler (same origin, no CORS)
package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Snider/Borg/pkg/player"
	"github.com/Snider/Borg/pkg/smsg"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend
var frontendAssets embed.FS

// MediaStore holds decrypted media in memory
type MediaStore struct {
	mu    sync.RWMutex
	media map[string]*MediaItem
}

type MediaItem struct {
	Data     []byte
	MimeType string
	Name     string
}

var globalStore = &MediaStore{media: make(map[string]*MediaItem)}

func (s *MediaStore) Set(id string, item *MediaItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.media[id] = item
}

func (s *MediaStore) Get(id string) *MediaItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.media[id]
}

func (s *MediaStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.media = make(map[string]*MediaItem)
}

// AssetHandler serves both static assets and decrypted media
type AssetHandler struct {
	assets fs.FS
}

func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	path = strings.TrimPrefix(path, "/")

	// Check if this is a media request
	if strings.HasPrefix(path, "media/") {
		id := strings.TrimPrefix(path, "media/")
		item := globalStore.Get(id)
		if item == nil {
			http.NotFound(w, r)
			return
		}

		// Serve with range support for seeking
		w.Header().Set("Content-Type", item.MimeType)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.Itoa(len(item.Data)))

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") {
			rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
			parts := strings.Split(rangeHeader, "-")
			start, _ := strconv.Atoi(parts[0])
			end := len(item.Data) - 1
			if len(parts) > 1 && parts[1] != "" {
				end, _ = strconv.Atoi(parts[1])
			}
			if end >= len(item.Data) {
				end = len(item.Data) - 1
			}
			if start > end || start >= len(item.Data) {
				http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
				return
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(item.Data)))
			w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(item.Data[start : end+1])
			return
		}

		http.ServeContent(w, r, item.Name, time.Time{}, bytes.NewReader(item.Data))
		return
	}

	// Serve static assets
	data, err := fs.ReadFile(h.assets, "frontend/"+path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type
	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(path, ".wasm"):
		w.Header().Set("Content-Type", "application/wasm")
	}

	w.Write(data)
}

// App wraps player functionality
type App struct {
	ctx    context.Context
	player *player.Player
}

func NewApp() *App {
	return &App{
		player: player.NewPlayer(),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.player.Startup(ctx)
}

// MediaResult holds URLs for playback
type MediaResult struct {
	Body        string            `json:"body"`
	Subject     string            `json:"subject,omitempty"`
	From        string            `json:"from,omitempty"`
	Attachments []MediaAttachment `json:"attachments,omitempty"`
}

type MediaAttachment struct {
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
	URL      string `json:"url"` // /media/0, /media/1, etc.
}

// LoadDemo decrypts demo and stores in memory for streaming
func (a *App) LoadDemo() (*MediaResult, error) {
	globalStore.Clear()

	// Read demo from embedded filesystem
	demoBytes, err := fs.ReadFile(frontendAssets, "frontend/demo-track.smsg")
	if err != nil {
		return nil, fmt.Errorf("demo not found: %w", err)
	}

	// Decrypt
	msg, err := smsg.Decrypt(demoBytes, "dapp-fm-2024")
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %w", err)
	}

	result := &MediaResult{
		Body:    msg.Body,
		Subject: msg.Subject,
		From:    msg.From,
	}

	for i, att := range msg.Attachments {
		// Decode base64 to raw bytes
		data, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			continue
		}

		// Store in memory
		id := strconv.Itoa(i)
		globalStore.Set(id, &MediaItem{
			Data:     data,
			MimeType: att.MimeType,
			Name:     att.Name,
		})

		result.Attachments = append(result.Attachments, MediaAttachment{
			Name:     att.Name,
			MimeType: att.MimeType,
			Size:     len(data),
			URL:      "/media/" + id,
		})
	}

	return result, nil
}

// GetDemoManifest returns manifest without decrypting
func (a *App) GetDemoManifest() (*player.ManifestInfo, error) {
	demoBytes, err := fs.ReadFile(frontendAssets, "frontend/demo-track.smsg")
	if err != nil {
		return nil, fmt.Errorf("demo not found: %w", err)
	}

	info, err := smsg.GetInfo(demoBytes)
	if err != nil {
		return nil, err
	}

	result := &player.ManifestInfo{}
	if info.Manifest != nil {
		m := info.Manifest
		result.Title = m.Title
		result.Artist = m.Artist
		result.Album = m.Album
		result.ReleaseType = m.ReleaseType
		result.Format = m.Format
		result.LicenseType = m.LicenseType

		for _, t := range m.Tracks {
			result.Tracks = append(result.Tracks, player.TrackInfo{
				Title:    t.Title,
				Start:    t.Start,
				End:      t.End,
				TrackNum: t.TrackNum,
			})
		}
	}

	return result, nil
}

// DecryptAndServe decrypts user-provided content and serves via asset handler
func (a *App) DecryptAndServe(encrypted string, password string) (*MediaResult, error) {
	globalStore.Clear()

	// Decrypt using player (handles base64 input)
	msg, err := smsg.DecryptBase64(encrypted, password)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %w", err)
	}

	result := &MediaResult{
		Body:    msg.Body,
		Subject: msg.Subject,
		From:    msg.From,
	}

	for i, att := range msg.Attachments {
		data, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			continue
		}

		id := strconv.Itoa(i)
		globalStore.Set(id, &MediaItem{
			Data:     data,
			MimeType: att.MimeType,
			Name:     att.Name,
		})

		result.Attachments = append(result.Attachments, MediaAttachment{
			Name:     att.Name,
			MimeType: att.MimeType,
			Size:     len(data),
			URL:      "/media/" + id,
		})
	}

	return result, nil
}

// Proxy methods
func (a *App) GetManifest(encrypted string) (*player.ManifestInfo, error) {
	return a.player.GetManifest(encrypted)
}

func (a *App) IsLicenseValid(encrypted string) (bool, error) {
	return a.player.IsLicenseValid(encrypted)
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "dapp.fm Player",
		Width:     1200,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Handler: &AssetHandler{assets: frontendAssets},
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
