package tim

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestNew(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if m == nil {
		t.Fatal("New() returned a nil tim")
	}
	if m.Config == nil {
		t.Error("New() returned a tim with a nil config")
	}
	if m.RootFS == nil {
		t.Error("New() returned a tim with a nil RootFS")
	}
	var js json.RawMessage
	if err := json.Unmarshal(m.Config, &js); err != nil {
		t.Error("New() returned a tim with invalid JSON config")
	}
}

func TestFromDataNode(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		dn := datanode.New()
		m, err := FromDataNode(dn)
		if err != nil {
			t.Fatalf("FromDataNode() error = %v", err)
		}
		if m == nil {
			t.Fatal("FromDataNode() returned a nil tim")
		}
	})

	t.Run("Bad", func(t *testing.T) {
		_, err := FromDataNode(nil)
		if !errors.Is(err, ErrDataNodeRequired) {
			t.Errorf("FromDataNode() with nil datanode should return ErrDataNodeRequired, got %v", err)
		}
	})
}

func TestToTar(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		dn := datanode.New()
		dn.AddData("test.txt", []byte("hello"))
		m, err := FromDataNode(dn)
		if err != nil {
			t.Fatalf("FromDataNode() error = %v", err)
		}
		_, err = m.ToTar()
		if err != nil {
			t.Fatalf("ToTar() error = %v", err)
		}
	})

	t.Run("Bad", func(t *testing.T) {
		m, _ := New()
		m.Config = nil
		_, err := m.ToTar()
		if !errors.Is(err, ErrConfigIsNil) {
			t.Errorf("ToTar() with nil config should return ErrConfigIsNil, got %v", err)
		}
	})
}
