package connect

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/ripnet/shellodex/internal/model"
)

// Telnet exec-replaces the current process with the system telnet binary.
// On success this function never returns.
func Telnet(host *model.Host) error {
	telnetPath, err := exec.LookPath("telnet")
	if err != nil {
		return fmt.Errorf("telnet not found in PATH: %w", err)
	}

	port := host.Port
	if port == 0 {
		port = 23
	}

	args := []string{"telnet", host.Hostname, fmt.Sprintf("%d", port)}

	return syscall.Exec(telnetPath, args, os.Environ())
}

// Connect dispatches to SSH or Telnet based on the host protocol.
// On success this function never returns.
func Connect(host *model.Host, cfg *model.Config) error {
	jumps, err := ResolveJumps(host, cfg)
	if err != nil {
		return err
	}

	cred := cfg.EffectiveCredential(host)

	// Best-effort: stash the password on the clipboard so the user can paste it
	// at ssh/telnet's prompt. Errors are intentionally ignored.
	if cred != nil && cred.Password != "" {
		_ = CopyPassword(cred.Password, cfg.ClipboardMode)
	}

	switch host.Protocol {
	case model.Telnet:
		return Telnet(host)
	default:
		return SSH(host, cred, jumps)
	}
}
