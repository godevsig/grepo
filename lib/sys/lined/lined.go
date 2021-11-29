// Package lined is line editor, supports operations on terminal.
//
// Supported operations includes short-cut like Ctrl-A Ctrl-E Ctrl-C Ctrl-D...
// and UP/DOWN for history.
package lined

import (
	"strings"

	"github.com/peterh/liner"
)

// Cfg is the options that lined supports, used when New the Editor instance.
type Cfg struct {
	Prompt string
}

// Editor is the instance of a line editor
type Editor struct {
	lnr *liner.State
	cfg Cfg
}

// NewEditor returns an instance of Editor, with user defined Cfg.
func NewEditor(cfg Cfg) *Editor {
	lnr := liner.NewLiner()
	lnr.SetMultiLineMode(true)
	return &Editor{lnr, cfg}
}

// Readline reads and returns a line from input not including
// a trailing newline character.
// Readline returns empty line if user only inputs a newline.
// An io.EOF error is returned if user entered Ctrl-D.
// Support multi line continuation ending with "\".
func (ed *Editor) Readline() (line string, err error) {
	p := ed.cfg.Prompt

	for {
		curline, err := ed.lnr.Prompt(p)
		if err != nil {
			return "", err
		}
		if len(strings.TrimSpace(curline)) == 0 {
			return "", nil
		}
		ed.lnr.AppendHistory(curline)
		if curline[len(curline)-1] != '\\' {
			line += curline
			break
		}
		p = "> "
		line += curline[:len(curline)-1] + "\n"
	}

	return line, nil
}

// Close returns the terminal to its previous mode.
func (ed *Editor) Close() {
	ed.lnr.Close()
}
