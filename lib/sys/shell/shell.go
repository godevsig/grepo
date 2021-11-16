package shell

import (
	"os"
	"os/exec"
	"strings"
)

// Run runs a command and returns its output.
// The command will be running in background and output is discarded
// if it ends with &.
func Run(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	bg := false
	if fields[len(fields)-1] == "&" {
		bg = true
		fields = fields[:len(fields)-1]
	}
	if _, err := os.Stat(fields[0]); os.IsNotExist(err) {
		fields = append([]string{"-c"}, strings.Join(fields, " "))
	}
	if bg {
		if err := exec.Command("sh", fields...).Start(); err != nil {
			return err.Error()
		}
		return ""
	}
	output, _ := exec.Command("sh", fields...).CombinedOutput()
	return string(output)
}
