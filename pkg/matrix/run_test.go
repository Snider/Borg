package matrix

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestRun_Good(t *testing.T) {
	// Create a dummy matrix file.
	file, err := os.CreateTemp("", "matrix-*.matrix")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	ExecCommand = fakeExecCommand
	defer func() { ExecCommand = exec.Command }()

	err = Run(file.Name())
	if err != nil {
		t.Errorf("Run() failed: %v", err)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	if cmd == "runc" && args[0] == "run" {
		fmt.Println("Success")
		os.Exit(0)
	} else {
		fmt.Fprintf(os.Stderr, "Unknown command %s %s\n", cmd, strings.Join(args, " "))
		os.Exit(1)
	}
}
