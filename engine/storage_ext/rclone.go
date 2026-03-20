package storageext

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
	Output(ctx context.Context, name string, args ...string) (string, error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ExecCommandRunner) Output(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run %s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

type RcloneRunner struct {
	Bin   string
	Exec  CommandRunner
	Extra []string
}

func NewRcloneRunner() RcloneRunner {
	bin := strings.TrimSpace(os.Getenv("RCLONE_BIN"))
	if bin == "" {
		bin = "rclone"
	}
	return RcloneRunner{
		Bin:  bin,
		Exec: ExecCommandRunner{},
	}
}

func (r RcloneRunner) Run(ctx context.Context, args ...string) error {
	if r.Exec == nil {
		r.Exec = ExecCommandRunner{}
	}
	fullArgs := append([]string{}, r.Extra...)
	fullArgs = append(fullArgs, args...)
	if err := r.Exec.Run(ctx, r.Bin, fullArgs...); err != nil {
		return fmt.Errorf("run %s %s: %w", r.Bin, strings.Join(fullArgs, " "), err)
	}
	return nil
}

func (r RcloneRunner) Output(ctx context.Context, args ...string) (string, error) {
	if r.Exec == nil {
		r.Exec = ExecCommandRunner{}
	}
	fullArgs := append([]string{}, r.Extra...)
	fullArgs = append(fullArgs, args...)
	out, err := r.Exec.Output(ctx, r.Bin, fullArgs...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r RcloneRunner) Delete(ctx context.Context, args ...string) error {
	if r.Exec == nil {
		r.Exec = ExecCommandRunner{}
	}
	fullArgs := append([]string{}, r.Extra...)
	fullArgs = append(fullArgs, args...)
	if err := r.Exec.Run(ctx, r.Bin, fullArgs...); err != nil {
		return fmt.Errorf("run %s %s: %w", r.Bin, strings.Join(fullArgs, " "), err)
	}
	return nil
}
