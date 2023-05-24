package shell

import (
	"io"
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
func (s Shell) Run(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", nil
	}
	bg := false
	if cmd[len(cmd)-1] == '&' {
		bg = true
		cmd = cmd[:len(cmd)-1]
	}

	if bg {
		return "", exec.Command("sh", "-c", cmd).Start()
	}
	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	return string(output), err
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
func Run(cmd string) (string, error) {
	return New("sh").Run(cmd)
}

// RunWith is shortcut for New("sh").RunWith(in, out)
func RunWith(in io.Reader, out io.Writer) error {
	return New("sh").RunWith(in, out)
}
