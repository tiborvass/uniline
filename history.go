package uniline

import (
	"bufio"
	"fmt"
	"os"

	"github.com/tiborvass/uniline/utils"
)

// ClearHistory Clears history.
func (s *Scanner) ClearHistory() {
	s.i.H = utils.History{}
}

// AddToHistory adds a string line to history
func (s *Scanner) AddToHistory(line string) {
	s.i.H.Tmp = append(s.i.H.Tmp, line)
	s.i.H.Saved = append(s.i.H.Saved, line)
}

// SaveHistory saves the current history to a file specified by filename.
func (s *Scanner) SaveHistory(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, line := range s.i.H.Tmp[:len(s.i.H.Tmp)-1] {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}
	return nil
}

// LoadHistory loads history from a file specified by filename.
func (s *Scanner) LoadHistory(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// tmp = saved = loaded History
	s.i.H.Saved = lines
	s.i.H.Tmp = make([]string, len(s.i.H.Saved))
	copy(s.i.H.Tmp, s.i.H.Saved)

	// add current line
	s.i.H.Tmp = append(s.i.H.Tmp, s.i.Buf.String())
	s.i.H.Index = len(s.i.H.Tmp) - 1
	return nil
}
