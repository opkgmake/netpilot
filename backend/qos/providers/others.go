// backend/qos/providers/others.go
package providers

import "strconv"

// ApplyTbf applies the Token Bucket Filter qdisc for simple rate limiting.
func ApplyTbf(ifaceName string, bandwidthMbit uint32) error {
	if err := DeleteRootQdisc(ifaceName); err != nil {
		return err
	}
	rate := strconv.FormatUint(uint64(bandwidthMbit), 10) + "mbit"

	// tbf requires a buffer size and limit, we use some sensible defaults.
	// Command: tc qdisc add dev <iface> root tbf rate <rate> buffer 1600 limit 3000
	output, err := runTcCommand("qdisc", "add", "dev", ifaceName, "root", "tbf", "rate", rate, "buffer", "1600", "limit", "3000")
	if err != nil {
		return formatTcError("failed to apply tbf qdisc", err, output)
	}
	return nil
}

// ApplyPfifoFast applies the system's default qdisc.
// This is effectively the same as deleting the root qdisc, as the kernel will reinstate pfifo_fast.
func ApplyPfifoFast(ifaceName string) error {
	return DeleteRootQdisc(ifaceName)
}

func ApplySfq(ifaceName string) error {
	if err := DeleteRootQdisc(ifaceName); err != nil {
		return err
	}
	output, err := runTcCommand("qdisc", "add", "dev", ifaceName, "root", "sfq")
	if err != nil {
		return formatTcError("failed to apply sfq qdisc", err, output)
	}
	return nil
}
