package telegram

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_Good(t *testing.T) {
	testDataPath := "testdata"

	dn, err := Parse(testDataPath)
	if err != nil {
		t.Fatalf("Parse() error = %v, wantErr nil", err)
	}

	if dn == nil {
		t.Fatal("Parse() returned a nil DataNode")
	}

	// Check for INDEX.json
	indexPath := "telegram/Test Channel/INDEX.json"
	exists, err := dn.Exists(indexPath)
	if err != nil {
		t.Fatalf("dn.Exists(%q) error: %v", indexPath, err)
	}
	if !exists {
		t.Errorf("Expected file to exist: %s", indexPath)
	}

	// Check for January messages markdown file
	janMessagesPath := "telegram/Test Channel/messages/2024-01.md"
	exists, err = dn.Exists(janMessagesPath)
	if err != nil {
		t.Fatalf("dn.Exists(%q) error: %v", janMessagesPath, err)
	}
	if !exists {
		t.Errorf("Expected file to exist: %s", janMessagesPath)
	} else {
		// Verify content of the January markdown file
		f, err := dn.Open(janMessagesPath)
		if err != nil {
			t.Fatalf("Failed to open %s: %v", janMessagesPath, err)
		}
		defer f.Close()

		contentBytes, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("Failed to read from %s: %v", janMessagesPath, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, "Hello, world!") {
			t.Errorf("Expected to find 'Hello, world!' in %s", janMessagesPath)
		}
		if !strings.Contains(content, "**This** is a *test* message with formatting.") {
			t.Errorf("Expected to find formatted message in %s", janMessagesPath)
		}
	}

	// Check for February messages markdown file
	febMessagesPath := "telegram/Test Channel/messages/2024-02.md"
	exists, err = dn.Exists(febMessagesPath)
	if err != nil {
		t.Fatalf("dn.Exists(%q) error: %v", febMessagesPath, err)
	}
	if !exists {
		t.Errorf("Expected file to exist: %s", febMessagesPath)
	} else {
		f, err := dn.Open(febMessagesPath)
		if err != nil {
			t.Fatalf("Failed to open %s: %v", febMessagesPath, err)
		}
		defer f.Close()

		contentBytes, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("Failed to read from %s: %v", febMessagesPath, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, "Here is a photo.") {
			t.Errorf("Expected to find 'Here is a photo.' in %s", febMessagesPath)
		}
	}

	// Check for media file
	mediaFileName := "photo_1@10-02-2024_18-30-00.jpg"
	mediaPath := filepath.Join("telegram", "Test Channel", "media", mediaFileName)
	mediaPath = filepath.ToSlash(mediaPath) // Ensure cross-platform path separators
	exists, err = dn.Exists(mediaPath)
	if err != nil {
		t.Fatalf("dn.Exists(%q) error: %v", mediaPath, err)
	}
	if !exists {
		t.Errorf("Expected media file to exist: %s", mediaPath)
	}
}
