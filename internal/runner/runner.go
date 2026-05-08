package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Invocation struct {
	Command []string
	Dir     string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

func Run(ctx context.Context, inv Invocation) (int, error) {
	if len(inv.Command) == 0 {
		return 2, fmt.Errorf("no command provided")
	}

	cmd := exec.CommandContext(ctx, inv.Command[0], inv.Command[1:]...)
	cmd.Dir = inv.Dir
	cmd.Env = inv.Env
	cmd.Stdin = inv.Stdin
	cmd.Stdout = inv.Stdout
	cmd.Stderr = inv.Stderr

	err := cmd.Run()
	if err == nil {
		return 0, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), err
	}
	if ctx.Err() != nil {
		return 130, ctx.Err()
	}
	if os.IsNotExist(err) {
		return 127, err
	}

	return 1, err
}
