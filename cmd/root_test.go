package cmd

import (
	"io"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecute(t *testing.T) {
	err := Execute(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}


func Test_NewRootCmd(t *testing.T) {
	if NewRootCmd() == nil {
		t.Errorf("NewRootCmd is nil")
	}
}
func Test_executeCommand(t *testing.T) {
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test with no args",
			args: args{
				cmd:  NewRootCmd(),
				args: []string{},
			},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args.cmd, tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
