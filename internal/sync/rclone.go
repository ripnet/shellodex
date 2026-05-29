package sync

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
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

// Pull copies the config file from the rclone remote to the local config directory.
// rclone copy treats the destination as a directory, so we pass the parent dir
// and rclone drops the file there by name.
func Pull(remote, localPath string) Result {
	return run("copy", remote, filepath.Dir(localPath))
}

// Sync mirrors the local config file to the remote (rclone sync, local → remote).
func Sync(localPath, remote string) Result {
	return run("sync", filepath.Dir(localPath), remote)
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
