//go:build !unix

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func execGh(gh string, args, env []string, stderr io.Writer) int {
	cmd := exec.Command(gh, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	if err != nil {
		fmt.Fprintln(stderr, "ghroute:", err)
		return 126
	}
	return 0
}
