package sync

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Result holds the outcome of an rclone operation.
type Result struct {
	Output string
	Err    error
}

// Push copies the local config file to the configured rclone remote.
func Push(localPath, remote string) Result {
	return run("copy", localPath, remote)
}

// Pull copies the config file from the rclone remote to the local path.
func Pull(remote, localPath string) Result {
	return run("copy", remote, localPath)
}

// Sync bidirectionally synchronizes (rclone bisync). The remote must be
// initialized with --resync on first use; this function uses --resync-mode
// newer to avoid prompting.
func Sync(localPath, remote string) Result {
	_, err := exec.LookPath("rclone")
	if err != nil {
		return Result{Err: fmt.Errorf("rclone not found in PATH: %w", err)}
	}
	cmd := exec.Command("rclone", "bisync", localPath, remote,
		"--resync-mode", "newer",
		"--conflict-resolve", "newer",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return Result{Output: out.String(), Err: err}
}

func run(op, src, dst string) Result {
	_, err := exec.LookPath("rclone")
	if err != nil {
		return Result{Err: fmt.Errorf("rclone not found in PATH: %w", err)}
	}
	cmd := exec.Command("rclone", op, src, dst, "--progress")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return Result{Output: out.String(), Err: err}
}
