package tim

import (
	"os"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestFromDataNode_Good(t *testing.T) {
	dn := setupDataNode(t)
	m, err := FromDataNode(dn)
	if err != nil {
		t.Fatalf("FromDataNode() error = %v", err)
	}
	if m == nil {
		t.Fatal("FromDataNode() returned a nil tim")
	}
	if m.RootFS != dn {
		t.Error("FromDataNode() did not set the RootFS correctly")
	}
}

func TestToTar_Good(t *testing.T) {
	m := setupTestTim(t)
	_, err := m.ToTar()
	if err != nil {
		t.Fatalf("ToTar() error = %v", err)
	}
}

// setupDataNode creates a simple DataNode for testing.
func setupDataNode(t *testing.T) *datanode.DataNode {
	t.Helper()
	dn := datanode.New()
	dn.AddData("test.txt", []byte("hello"))
	return dn
}

// setupTestTim creates a simple TerminalIsolationMatrix for testing.
func setupTestTim(t *testing.T) *TerminalIsolationMatrix {
	t.Helper()
	m, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	m.RootFS = setupDataNode(t)

	return m
}
