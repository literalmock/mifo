package metrics

import (
	"context"
	"os"
	"os/exec"
)

// command builds an exec.Cmd with a fixed C locale so that tool output
// (dates, decimal separators) parses identically on every Mac.
func command(ctx context.Context, name string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, name, args...)
	c.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	return c
}
