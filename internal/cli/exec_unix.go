//go:build unix

package cli

import (
	"fmt"
	"io"
	"syscall"
)

func execGh(gh string, args, env []string, stderr io.Writer) int {
	err := syscall.Exec(gh, append([]string{gh}, args...), env)
	fmt.Fprintln(stderr, "ghroute:", err)
	return 126
}
