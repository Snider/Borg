package tim

import (
	"errors"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

func TestNew_Error(t *testing.T) {
	origDefaultConfig := defaultConfigVar
	t.Cleanup(func() {
		defaultConfigVar = origDefaultConfig
	})

	// Test error from defaultConfigVar
	defaultConfigVar = func() (map[string]interface{}, error) {
		return nil, errors.New("mock defaultConfig error")
	}
	_, err := New()
	if err == nil {
		t.Fatal("Expected error from defaultConfig, got nil")
	}

	// Test error from json.Marshal
	defaultConfigVar = func() (map[string]interface{}, error) {
		return map[string]interface{}{"foo": make(chan int)}, nil
	}
	_, err = New()
	if err == nil {
		t.Fatal("Expected error from json.Marshal, got nil")
	}
}

func TestFromDataNode_Error(t *testing.T) {
	origDefaultConfig := defaultConfigVar
	t.Cleanup(func() {
		defaultConfigVar = origDefaultConfig
	})

	defaultConfigVar = func() (map[string]interface{}, error) {
		return nil, errors.New("mock defaultConfig error")
	}

	dn := datanode.New()
	_, err := FromDataNode(dn)
	if err == nil {
		t.Fatal("Expected error from FromDataNode, got nil")
	}
}
