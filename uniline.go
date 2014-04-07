package uniline

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/dotcloud/docker/pkg/term"
	. "github.com/tiborvass/file.debug"
	"github.com/tiborvass/uniline/ansi"
	"github.com/tiborvass/uniline/internals"
	"github.com/tiborvass/uniline/keymap"
	"github.com/tiborvass/uniline/utils"
)

func defaultOnInterrupt(s *Scanner) (more bool) {
	s.i.Output.Write([]byte("^C"))
	if len(s.i.Buf.Bytes) == 0 {
		os.Exit(1)
	}
	s.i.Buf = utils.Text{}
	return true
}

// Scanner provides a simple interface to read and, if possible, interactively edit a line using Ansi commands.
type Scanner struct {
	onInterrupt func(*Scanner) (more bool)
	km          keymap.Keymap
	i           *internals.Internals
}

type blackhole struct{}

var devNull = new(blackhole)

func (*blackhole) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// DefaultScanner returns a ready-to-use default Scanner.
//
// The input is set to os.Stdin (which is also the output if it's a TTY).
// On Ctrl-C, if the current line is empty, the program exits with status code 1, otherwise returns an empty line.
// The default keymap is the one available in package uniline/keymap.
//
// Note: equivalent to NewScanner(nil, nil, nil, nil)
func DefaultScanner() *Scanner {
	return NewScanner(nil, os.Stdout, nil, nil)
}

// NewScanner returns a ready-to-use Scanner with configurable settings.
//
// NewScanner also detects if ANSI-mode is available to let the user edit the input line. If it is not available, it falls back to a dumb-mode
// where scanning is using directly a bufio.Scanner using bufio.ScanLines.
//
// Any parameter can be nil in which case the defaults are used (c.f. DefaultScanner).
//
// In order to have a good line editing experience, input should be an *os.File with the same file descriptor as output
func NewScanner(input io.Reader, output io.Writer, onInterrupt func(s *Scanner) (more bool), km keymap.Keymap) *Scanner {
	if input == nil {
		input = os.Stdin
	}
	if onInterrupt == nil {
		onInterrupt = defaultOnInterrupt
	}
	if km == nil {
		km = keymap.DefaultKeymap()
	}

	s := &Scanner{onInterrupt, km, &internals.Internals{Input: input, Output: devNull, Dumb: true}}

	f, ok := input.(*os.File)
	if !ok {
		return s
	}

	if output != nil {
		_, ok := output.(*os.File)
		if !ok {
			return s
		}
	}
	s.i.Output = input.(io.Writer) // does not panic, since *os.File implements io.Writer

	fd := f.Fd()
	s.i.Fd = &fd
	t := os.Getenv("TERM")
	s.i.Dumb = !term.IsTerminal(fd) || len(t) == 0 || t == "dumb" || t == "cons25"
	return s
}

// Scan reads a line from the provided input and makes it available via Scanner.Bytes() and Scanner.Text().
// It returns a boolean indicating whether there can be more lines retrieved or if scanning has ended.
//
// Scanning can end either normally or with an error. The error will be available in Scanner.Err().
//
// If the input source (Scanner.Input) is a TTY, the line is editable, otherwise each line is returned.
// Upon Ctrl-C, the current input stops being scanned and Scanner.onInterrupt() whose boolean return value determines whether or not
// scanning should be completely aborted (more = false) or if only the current line should be discarded (more = true), accepting more scans.
func (s *Scanner) Scan(prompt string) (more bool) {
	defer func() {
		// dumb terminals have already printed newline
		if !s.i.Dumb {
			fmt.Fprintln(s.i.Output)
		}
	}()

	defer func() {
		if x := recover(); x != nil {
			var ok bool
			s.i.Err, ok = x.(error)
			if ok {
				// abort scanning because of an encountered error
				more = false
				return
			}
			if sig, ok := x.(os.Signal); ok && sig == os.Interrupt {
				// TODO: reconcile Signals and errors somehow, I don't like having "no error" on a SIGINT.
				s.i.Err = nil
				if s.onInterrupt == nil {
					s.onInterrupt = defaultOnInterrupt
				}
				more = s.onInterrupt(s)
				return
			}
			panic(x)
		}
	}()

	// no need to initialize internal scanner more than once
	if s.i.S == nil {
		s.i.S = bufio.NewScanner(s.i.Input)
	}

	s.i.Prompt = utils.TextFromString(prompt)
	s.i.Stop = false

	if s.i.Dumb {
		s.i.S.Split(bufio.ScanLines)

		if _, err := fmt.Fprint(s.i.Output, string(s.i.Prompt.Bytes)); err != nil {
			panic(err)
		}

		if !s.i.S.Scan() {
			return false
		}
		// note: buf is of type utils.Text, but only "bytes" is used when no tty.
		s.i.Buf.Bytes = s.i.S.Bytes()

		s.i.Err = s.i.S.Err()
		// continue scanning if no error
		return s.i.Err == nil
	}
	state, err := term.MakeRaw(*s.i.Fd)
	if err != nil {
		panic(err)
	}
	defer func() {
		term.RestoreTerminal(*s.i.Fd, state)
	}()
	winsize, err := term.GetWinsize(*s.i.Fd)
	if err != nil {
		panic(err)
	}

	s.i.Buf = utils.Text{}
	s.i.Pos = utils.Position{}
	s.i.Cols = int(winsize.Width)

	// create new empty temporary element in History
	s.i.H.Tmp = append(s.i.H.Tmp, "")
	// set History Index to this newly created empty element
	s.i.H.Index = len(s.i.H.Tmp) - 1

	s.i.Output.Write(s.i.Prompt.Bytes)
	s.i.S.Split(bufio.ScanRunes)

	var p []byte

	for !s.i.Stop && s.i.S.Scan() {

		var r rune

		var isCompleteAnsiCode = func() (done bool) {
			key := ansi.Code(p)
			scanFun, ok := s.km[key]
			if ok {
				if scanFun == nil {
					return false
				}
				scanFun(s.i)
			}
			return true
		}

		if p == nil {
			// In case where p is either one-rune long or it is the first byte of a long command

			p = s.i.S.Bytes()
			r = getRune(string(p))

			// if printable, then it's not a command
			if unicode.IsPrint(r) {
				s.i.Insert(utils.CharFromRune(r))
				// moving on to next rune
				p = nil
				continue
			}

			Debug("r: %v", r)

			if isCompleteAnsiCode() {
				// moving on to next rune
				p = nil
			}

			// handle special case for Clipboard
			if r != 23 && r != 21 && r != 11 {
				// not Ctrl-W, Ctrl-U, or Ctrl-K
				// thus consider the Clipboard as complete and stop gluing Clipboard parts together
				s.i.Cb.Partial = false
			}

		} else {
			// In the case where p is an escape sequence, add current bytes to previous and try a lookup

			p = append(p, s.i.S.Bytes()...)
			Debug("p: %v", p)
			if isCompleteAnsiCode() {
				p = nil
				Debug("done")
			}
		}
	}

	s.i.Err = s.i.S.Err()
	// if EOF, we need to consider last line
	if !s.i.Stop {
		s.i.Enter()
		return false
	}
	return s.i.Err == nil
}

// Err returns the first non-EOF error that was encountered by the Scanner.
func (s *Scanner) Err() error {
	return s.i.Err
}

// utils.Text returns the most recent line read from s.i.Input during a call to Scan as a newly allocated string holding its bytes.
func (s *Scanner) Text() string {
	return string(s.i.Buf.Bytes)
}

// Bytes returns the most recent line read from s.i.Input during a call to Scan.
// The underlying array may point to data that will be overwritten by subsequent call to Scan.
// It does no allocation.
func (s *Scanner) Bytes() []byte {
	return s.i.Buf.Bytes
}

// Internals exposes the unexported fields of Scanner.
// Useful when making a custom Keymap
func (s *Scanner) Internals() *internals.Internals {
	return s.i
}

// trick to get the first rune of a string without utf8 package
func getRune(str string) rune {
	var i int
	var r rune
	for i, r = range str {
		if i > 0 {
			panic("ScanRunes is supposed to scan one rune at a time, but received more than one")
		}
	}
	return r
}
