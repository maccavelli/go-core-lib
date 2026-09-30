package selfupdate

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

var isTerminal = term.IsTerminal

type terminalConfirmer struct {
	in  *os.File
	out io.Writer

	// One reader goroutine per confirmer delivers lines through lines, so a
	// Confirm cancelled mid-read leaves its line for the next Confirm rather
	// than losing it (0003-MADR C7).
	start sync.Once
	lines chan lineResult
}

type lineResult struct {
	line string
	err  error
}

// NewTerminalConfirmer prompts on out and reads from in. A non-terminal input
// returns ErrConfirmationRequired instead of hanging. End of input before an
// answer is a decline. A cancelled Confirm returns the context error; a line
// typed afterwards answers the next Confirm on the same confirmer.
func NewTerminalConfirmer(in *os.File, out io.Writer) Confirmer {
	return &terminalConfirmer{in: in, out: out}
}

func (c *terminalConfirmer) readLines() {
	c.lines = make(chan lineResult, 1)
	go func() {
		s := bufio.NewScanner(c.in)
		for s.Scan() {
			c.lines <- lineResult{line: s.Text()}
		}
		err := s.Err()
		if err == nil {
			err = io.EOF
		}
		c.lines <- lineResult{err: err}
		close(c.lines)
	}()
}

func (c *terminalConfirmer) Confirm(ctx context.Context, p Prompt) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if c.in == nil {
		return false, fmt.Errorf("selfupdate: confirmation input is nil: %w", ErrConfirmationRequired)
	}
	if !isTerminal(int(c.in.Fd())) {
		return false, fmt.Errorf("selfupdate: pass --yes to apply without a TTY: %w", ErrConfirmationRequired)
	}
	if c.out == nil {
		return false, fmt.Errorf("selfupdate: confirmation output is nil")
	}
	prompt := fmt.Sprintf("selfupdate: %s %s from %s to %s? [y/N] ",
		sanitizeText(p.Operation.String()),
		sanitizeText(p.Product),
		sanitizeText(p.Current),
		sanitizeText(p.Target),
	)
	if _, err := io.WriteString(c.out, prompt); err != nil {
		return false, err
	}
	c.start.Do(c.readLines)
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case r, ok := <-c.lines:
		if !ok || errors.Is(r.err, io.EOF) {
			// End of input before an answer is the default: decline.
			return false, nil
		}
		if r.err != nil {
			return false, r.err
		}
		switch strings.ToLower(strings.TrimSpace(r.line)) {
		case "y", "yes":
			return true, nil
		default:
			return false, nil
		}
	}
}
