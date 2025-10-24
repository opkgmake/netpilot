package providers

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// runTcCommand executes the `tc` command with the provided arguments while handling
// privilege escalation when necessary. It first attempts to run the command
// directly. When NetPilot is not running as root, the helper retries with
// common privilege escalation tools (sudo/doas) if they are available.
//
// The function returns the combined standard output and error streams along
// with the execution error (if any). Callers can inspect both values for
// richer error messages.
func runTcCommand(args ...string) ([]byte, error) {
	cmd := exec.Command("tc", args...)
	output, err := cmd.CombinedOutput()
	if err == nil || os.Geteuid() == 0 {
		return output, err
	}

	var lastOutput = output
	var lastErr = err

	for _, wrapper := range []string{"sudo", "doas"} {
		if _, lookupErr := exec.LookPath(wrapper); lookupErr == nil {
			cmd = exec.Command(wrapper, append([]string{"tc"}, args...)...)
			lastOutput, lastErr = cmd.CombinedOutput()
			if lastErr == nil {
				return lastOutput, nil
			}
		}
	}

	return lastOutput, lastErr
}

// formatTcError enriches the error returned by runTcCommand with additional
// context and actionable guidance.
func formatTcError(action string, err error, output []byte) error {
	if err == nil {
		return nil
	}

	trimmed := strings.TrimSpace(string(output))
	var builder strings.Builder
	builder.WriteString(action)
	builder.WriteString(": ")

	// Provide a clearer hint when privilege escalation is the likely cause.
	var execErr *exec.ExitError
	if errors.As(err, &execErr) {
		if trimmed == "" {
			trimmed = execErr.Error()
		}
		if strings.Contains(strings.ToLower(trimmed), "operation not permitted") {
			builder.WriteString("insufficient privileges. Run NetPilot as root or install sudo/doas. ")
		}
	}

	builder.WriteString("command failed: ")
	builder.WriteString(err.Error())
	if trimmed != "" {
		builder.WriteString(", output: ")
		builder.WriteString(trimmed)
	}

	return errors.New(builder.String())
}
