package connect

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/ripnet/shellodex/internal/model"
)

// SSH builds the ssh argv and exec-replaces the current process.
// On success this function never returns.
func SSH(host *model.Host, cred *model.Credential, jumps []string) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}

	args := []string{"ssh"}

	if cred != nil && cred.Username != "" {
		args = append(args, "-l", cred.Username)
	}

	port := host.Port
	if port == 0 {
		port = 22
	}
	if port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", port))
	}

	if cred != nil && cred.KeyPath != "" {
		args = append(args, "-i", cred.KeyPath)
	}

	if len(jumps) > 0 {
		args = append(args, "-J", strings.Join(jumps, ","))
	}

	args = append(args, host.Hostname)

	return syscall.Exec(sshPath, args, os.Environ())
}

// SSHArgs returns the argv that would be passed to ssh, for display purposes.
func SSHArgs(host *model.Host, cred *model.Credential, jumps []string) []string {
	args := []string{"ssh"}
	if cred != nil && cred.Username != "" {
		args = append(args, "-l", cred.Username)
	}
	port := host.Port
	if port == 0 {
		port = 22
	}
	if port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", port))
	}
	if cred != nil && cred.KeyPath != "" {
		args = append(args, "-i", cred.KeyPath)
	}
	if len(jumps) > 0 {
		args = append(args, "-J", strings.Join(jumps, ","))
	}
	args = append(args, host.Hostname)
	return args
}
