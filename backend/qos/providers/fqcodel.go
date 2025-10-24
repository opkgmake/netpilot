// backend/qos/providers/fqcodel.go
package providers

// No extra imports required.

// ApplyFqCodel applies the fq_codel qdisc.
// NOTE: fq_codel itself doesn't have a bandwidth parameter.
// For real-world use, it's often paired with another qdisc like HTB for rate limiting.
// For this MVP, we will apply it directly, which is useful for solving bufferbloat
// on links that are already shaped by the ISP.
func ApplyFqCodel(ifaceName string) error {
	// First, clean up any existing root qdisc.
	if err := DeleteRootQdisc(ifaceName); err != nil {
		return err
	}

	// Run the command (with automatic privilege escalation when possible).
	output, err := runTcCommand("qdisc", "add", "dev", ifaceName, "root", "fq_codel")
	if err != nil {
		return formatTcError("failed to apply fq_codel qdisc", err, output)
	}

	return nil
}
