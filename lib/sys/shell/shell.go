package shell

import (
	"io"
	"os"
	"os/exec"
	"strings"
)

// Shell is shell.
type Shell struct {
	sh string
}

// New creates a new shell, sh can be "sh" or "bash"
// or anyother system supported shell.
func New(sh string) Shell {
	return Shell{sh}
}

// Run runs a file or a command and returns its output.
// The command will be running in background and its output is discarded
// if it ends with &.
func (s Shell) Run(cmd string) string {
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
		if err := exec.Command(s.sh, fields...).Start(); err != nil {
			return err.Error()
		}
		return ""
	}
	output, _ := exec.Command(s.sh, fields...).CombinedOutput()
	return string(output)
}

// RunWith reads and executes whatever from in and outputs to out.
func (s Shell) RunWith(in io.Reader, out io.Writer) error {
	cmd := exec.Command(s.sh, "-i")
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// Run is shortcut for New("sh").Run(cmd)
func Run(cmd string) string {
	return New("sh").Run(cmd)
}

// RunWith is shortcut for New("sh").RunWith(in, out)
func RunWith(in io.Reader, out io.Writer) error {
	return New("sh").RunWith(in, out)
}
