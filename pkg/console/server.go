package console

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Borg/pkg/tim"
)

//go:embed unlock.html
var unlockHTML []byte

// Server serves encrypted STIM content with an optional unlock page.
type Server struct {
	stimData []byte
	password string
	port     string

	mu       sync.RWMutex
	unlocked bool
	rootFS   *datanode.DataNode
}

// NewServer creates a new console server.
// If password is provided, the content is decrypted immediately.
// If password is empty, an unlock page is shown until the user provides the password.
func NewServer(stimPath, password, port string) (*Server, error) {
	data, err := os.ReadFile(stimPath)
	if err != nil {
		return nil, fmt.Errorf("reading STIM file: %w", err)
	}

	s := &Server{
		stimData: data,
		password: password,
		port:     port,
	}

	// If password provided, unlock immediately
	if password != "" {
		if err := s.unlock(password); err != nil {
			return nil, fmt.Errorf("decrypting STIM: %w", err)
		}
	}

	return s, nil
}

// unlock decrypts the STIM data with the given password.
func (s *Server) unlock(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := tim.FromSigil(s.stimData, password)
	if err != nil {
		return err
	}

	s.rootFS = m.RootFS
	s.unlocked = true
	return nil
}

// isUnlocked returns whether the content has been decrypted.
func (s *Server) isUnlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unlocked
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	http.HandleFunc("/", s.handleRoot)
	http.HandleFunc("/unlock", s.handleUnlock)

	return http.ListenAndServe(":"+s.port, nil)
}

// handleRoot serves the main content or unlock page.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if !s.isUnlocked() {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(unlockHTML)
		return
	}

	s.mu.RLock()
	fs := http.FS(s.rootFS)
	s.mu.RUnlock()

	http.FileServer(fs).ServeHTTP(w, r)
}

// handleUnlock processes the unlock form submission.
func (s *Server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "Invalid form submission")
		return
	}

	password := r.FormValue("password")
	if password == "" {
		redirectWithError(w, r, "Password is required")
		return
	}

	if err := s.unlock(password); err != nil {
		redirectWithError(w, r, "Incorrect password")
		return
	}

	// Success - redirect to content
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// redirectWithError redirects to the unlock page with an error message.
func redirectWithError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, "/?error="+url.QueryEscape(message), http.StatusSeeOther)
}

// Port returns the server's port.
func (s *Server) Port() string {
	return s.port
}

// URL returns the full server URL.
func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%s", s.port)
}
