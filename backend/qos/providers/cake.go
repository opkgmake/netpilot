package providers

import (
	"strconv"
	"strings"
)

// ApplyCake applies the CAKE qdisc by executing the 'tc' command.
func ApplyCake(ifaceName string, bandwidthMbit uint32) error {
	// First, try to delete any existing root qdisc to avoid conflicts.
	if err := DeleteRootQdisc(ifaceName); err != nil {
		return err
	}

	// Convert bandwidth to a string like "100mbit"
	rate := strconv.FormatUint(uint64(bandwidthMbit), 10) + "mbit"

	// Run the command (with automatic privilege escalation when possible).
	output, err := runTcCommand("qdisc", "add", "dev", ifaceName, "root", "cake", "bandwidth", rate)
	if err != nil {
		return formatTcError("failed to apply cake qdisc", err, output)
	}

	return nil
}

// DeleteRootQdisc removes the root qdisc by executing the 'tc' command.
func DeleteRootQdisc(ifaceName string) error {
	output, err := runTcCommand("qdisc", "del", "dev", ifaceName, "root")
	if err != nil {
		// Silently ignore the "qdisc not found" error, but surface everything else.
		msg := strings.ToLower(string(output))
		if strings.Contains(msg, "no such file or directory") {
			return nil
		}
		return formatTcError("failed to delete existing qdisc", err, output)
	}
	return nil
}
