package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// ConfirmCommands controls whether Connect installs a confirmation gate that
// prompts on stdin before every command runs on a host. It is set from the
// global --confirm CLI flag and mirrors the behaviour of rig v1's exec.Confirm.
var ConfirmCommands bool

// confirmMu serializes prompts so that parallel host operations do not
// interleave their questions on the shared stdin/stderr.
var confirmMu sync.Mutex

// confirmCommand prompts on stderr for approval of command on host and reads the
// answer from stdin. It returns true when the command may run (an empty answer
// or "y"/"yes" allows it). The command is the fully decorated, redacted form
// that rig is about to execute.
func confirmCommand(host, command string) bool {
	confirmMu.Lock()
	defer confirmMu.Unlock()

	fmt.Fprintf(os.Stderr, "\nHost: %s\nCommand: %s\nAllow? [Y/n]: ", host, command)

	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		// Treat an unreadable stdin (e.g. EOF from a closed pipe) as a refusal
		// so we never run an unconfirmed command.
		return false
	}
	answer = strings.TrimSpace(answer)

	return answer == "" || strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}
