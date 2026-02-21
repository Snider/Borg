package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestProgressFromCmd_Good(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().BoolP("quiet", "q", false, "")

	p := ProgressFromCmd(cmd)
	if p == nil {
		t.Fatal("expected non-nil Progress")
	}
}

func TestProgressFromCmd_Quiet_Good(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().BoolP("quiet", "q", true, "")
	_ = cmd.PersistentFlags().Set("quiet", "true")

	p := ProgressFromCmd(cmd)
	if p == nil {
		t.Fatal("expected non-nil Progress")
	}
}
