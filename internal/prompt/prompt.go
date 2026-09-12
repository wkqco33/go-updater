// Package prompt handles interactive confirmation so destructive commands can
// stay safe in a terminal and explicit in scripts and agents.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrInputRequired reports that a confirmation could not be asked because no
// interactive terminal is available. Callers surface it as a usage error.
var ErrInputRequired = errors.New("confirmation requires an interactive terminal")

// Options configures one confirmation request.
type Options struct {
	In  io.Reader
	Out io.Writer
	// AssumeYes approves without asking (--yes).
	AssumeYes bool
	// NoInput forbids prompting (--no-input).
	NoInput bool
	// IsTTY overrides terminal detection. Nil uses IsTerminal(In).
	IsTTY func(io.Reader) bool
}

// Confirm asks the user to approve a destructive action. It returns false for
// an explicit or empty answer (cancelled), and ErrInputRequired when the
// environment cannot answer the prompt at all.
func Confirm(opts Options, question, approveFlag string) (bool, error) {
	if opts.AssumeYes {
		return true, nil
	}
	if opts.NoInput || !isTTY(opts) {
		return false, fmt.Errorf("%w: %s로 비대화형 실행을 승인하세요", ErrInputRequired, approveFlag)
	}

	fmt.Fprintf(opts.Out, "%s [y/N]: ", question)
	scanner := bufio.NewScanner(opts.In)
	if !scanner.Scan() {
		return false, fmt.Errorf("%w: 표준 입력에서 답변을 읽지 못했습니다", ErrInputRequired)
	}
	return strings.EqualFold(strings.TrimSpace(scanner.Text()), "y"), nil
}

func isTTY(opts Options) bool {
	if opts.IsTTY != nil {
		return opts.IsTTY(opts.In)
	}
	return IsTerminal(opts.In)
}

// IsTerminal reports whether r is an interactive terminal. Pipes and regular
// files report false; a character device such as /dev/null reports true, in
// which case Confirm fails safely on EOF.
func IsTerminal(r io.Reader) bool {
	file, ok := r.(*os.File)
	return ok && isCharDevice(file)
}

// IsTerminalWriter is the io.Writer counterpart of IsTerminal, used to decide
// whether human-facing output may include color or progress.
func IsTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && isCharDevice(file)
}

func isCharDevice(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
