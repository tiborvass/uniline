package internals

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/tiborvass/uniline/ansi"
	"github.com/tiborvass/uniline/utils"
)

// Data structure of the internals of Scanner, useful when creating a custom Keymap.
type Internals struct {
	Input  io.Reader
	Output io.Writer
	S      *bufio.Scanner
	H      History
	Prompt utils.Text
	Cb     Clipboard
	Pos    utils.Position
	Cols   int // number of columns, aka window width
	Buf    utils.Text
	Err    error // the error that will be returned in Err()
	Dumb   bool
	Fd     *uintptr

	// Whether to stop current line's scanning
	// This is used for internal scanning.
	// Termination of external scanning is handled with the boolean return variable `more`
	Stop bool
}

type Clipboard struct {
	Text    utils.Text
	Partial bool
}

type History struct {
	Saved []string
	Tmp   []string
	Index int
}

func (i *Internals) Insert(c utils.Char) {
	if i.Buf.ColLen == i.Pos.Columns {
		i.Buf = i.Buf.AppendChar(c)
		i.Pos = i.Pos.Add(c)
		if i.Prompt.ColLen+i.Buf.ColLen < i.Cols { // TODO: handle multiline
			mustWrite(i.Output.Write(c.P))
		} else {
			i.Refresh()
		}
	} else {
		i.Buf = i.Buf.InsertCharAt(i.Pos, c)
		i.Pos = i.Pos.Add(c)
		i.Refresh()
	}
}

func (i *Internals) Enter() {
	// removing most recent element of History
	// if user actually wants to add it, he can call Scanner.AddToHistory(line)
	i.H.Tmp = i.H.Tmp[:len(i.H.Tmp)-1]

	// Note: Design decision (differs from the readline in bash)
	//
	// > foo⏎ 			(yields "foo" + assuming it is added to History)
	// > bar⏎ 			(yields "bar" + assuming it is added to History)
	// > ↑				(going 1 element back in History)
	// > bar
	// > bar2 			(modifying bar element; note that Enter was not hit)
	// > ↑				(going 1 element back in History)
	// > foo
	// > foo42			(modifying foo element)
	// > foo42⏎			(hitting Enter, yields "foo42" + assuming it is added to History)
	//
	// At the end, History looks like ["foo", "bar", "foo42"] losing "bar2".
	// This is differing from bash where History would look like ["foo", "bar2", "foo42"] losing "bar".
	copy(i.H.Tmp, i.H.Saved)
	i.Stop = true
}

func (i *Internals) Interrupt() {
	i.H.Tmp = i.H.Tmp[:len(i.H.Tmp)-1]
	panic(os.Interrupt)
}

func (i *Internals) DeleteOrEOF() {
	if len(i.Buf.Chars) == 0 {
		var err error
		// since err is of type error and is nil, it will result in a clean EOF
		// look at the defer in Scan() in uniline.go
		panic(err)
	}
	i.Delete()
}

func (i *Internals) Backspace() {
	if i.Pos.Runes > 0 && len(i.Buf.Chars) > 0 {
		c := i.Buf.Chars[i.Pos.Runes-1]
		pos2 := i.Pos.Subtract(c)
		i.Buf = i.Buf.RemoveCharAt(pos2)
		i.Pos = pos2
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) Delete() {
	if len(i.Buf.Chars) > 0 && i.Pos.Runes < len(i.Buf.Chars) {
		i.Buf = i.Buf.RemoveCharAt(i.Pos)
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) MoveLeft() {
	if i.Pos.Runes > 0 {
		i.Pos = i.Pos.Subtract(i.Buf.Chars[i.Pos.Runes-1])
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) MoveRight() {
	if i.Pos.Runes < len(i.Buf.Chars) {
		i.Pos = i.Pos.Add(i.Buf.Chars[i.Pos.Runes])
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) MoveWordLeft() {
	if i.Pos.Runes > 0 {
		var nonSpaceEncountered bool
		for pos := i.Pos.Runes - 1; pos >= 0; pos-- {
			c := i.Buf.Chars[pos]
			if unicode.IsSpace(c.R) {
				if nonSpaceEncountered {
					break
				}
			} else if !nonSpaceEncountered {
				nonSpaceEncountered = true
			}
			i.Pos = i.Pos.Subtract(c)
		}
		i.Refresh()
	}
}

func (i *Internals) MoveWordRight() {
	if i.Pos.Runes < len(i.Buf.Chars) {
		var nonSpaceEncountered bool
		for pos := i.Pos.Runes; pos < len(i.Buf.Chars); pos++ {
			c := i.Buf.Chars[pos]
			if unicode.IsSpace(c.R) {
				if nonSpaceEncountered {
					break
				}
			} else if !nonSpaceEncountered {
				nonSpaceEncountered = true
			}
			i.Pos = i.Pos.Add(c)
		}
		i.Refresh()
	}
}

func (i *Internals) MoveBeginning() {
	i.Pos = utils.Position{}
	i.Refresh()
}

func (i *Internals) MoveEnd() {
	i.Pos = utils.Position{len(i.Buf.Chars), len(i.Buf.Bytes), i.Buf.ColLen}
	i.Refresh()
}

func (i *Internals) HistoryBack() {
	if i.H.Index > 0 {
		i.H.Tmp[i.H.Index] = i.Buf.String()
		i.H.Index--
		i.Buf = utils.TextFromString(i.H.Tmp[i.H.Index])
		i.Pos = utils.Position{len(i.Buf.Chars), len(i.Buf.Bytes), i.Buf.ColLen}
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) HistoryForward() {
	if i.H.Index < len(i.H.Tmp)-1 {
		i.H.Tmp[i.H.Index] = i.Buf.String()
		i.H.Index++
		i.Buf = utils.TextFromString(i.H.Tmp[i.H.Index])
		i.Pos = utils.Position{len(i.Buf.Chars), len(i.Buf.Bytes), i.Buf.ColLen}
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) CutLineLeft() {
	if i.Pos.Runes > 0 {
		if i.Cb.Partial {
			i.Cb.Text = i.Buf.Slice(utils.Position{}, i.Pos).AppendText(i.Cb.Text)
		} else {
			i.Cb.Text = i.Buf.Slice(utils.Position{}, i.Pos)
		}
		i.Buf = i.Buf.Slice(i.Pos)
		i.Pos = utils.Position{}
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) CutLineRight() {
	if i.Pos.Runes < len(i.Buf.Chars) {
		if i.Cb.Partial {
			i.Cb.Text = i.Cb.Text.AppendText(i.Buf.Slice(i.Pos).Clone())
		} else {
			i.Cb.Text = i.Buf.Slice(i.Pos).Clone()
		}
		i.Cb.Partial = true
		i.Buf = i.Buf.Slice(utils.Position{}, i.Pos)
		i.Refresh()
	}
}

func (i *Internals) CutPrevWord() {
	if i.Pos.Runes > 0 {
		pos := i.Pos
		var nonSpaceEncountered bool
		for pos.Runes > 0 {
			if unicode.IsSpace(i.Buf.Chars[pos.Runes-1].R) {
				if nonSpaceEncountered {
					break
				}
			} else if !nonSpaceEncountered {
				nonSpaceEncountered = true
			}
			pos = pos.Subtract(i.Buf.Chars[pos.Runes-1])
		}
		if i.Cb.Partial {
			i.Cb.Text = i.Buf.Slice(pos, i.Pos).Clone().AppendText(i.Cb.Text)
		} else {
			i.Cb.Text = i.Buf.Slice(pos, i.Pos).Clone()
		}
		i.Cb.Partial = true
		i.Buf = i.Buf.Slice(utils.Position{}, pos).AppendText(i.Buf.Slice(i.Pos))
		i.Pos = pos
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) SwapChars() {
	if i.Pos.Runes > 0 && len(i.Buf.Chars) > 1 {
		pos := i.Pos
		if i.Pos.Runes == len(i.Buf.Chars) {
			pos = pos.Subtract(i.Buf.Chars[i.Pos.Runes-1])
		}
		i.Buf.Chars[pos.Runes-1], i.Buf.Chars[pos.Runes] = i.Buf.Chars[pos.Runes], i.Buf.Chars[pos.Runes-1]
		i.Buf.Bytes[pos.Bytes-1], i.Buf.Bytes[pos.Bytes] = i.Buf.Bytes[pos.Bytes], i.Buf.Bytes[pos.Bytes-1]
		i.Pos = pos.Add(i.Buf.Chars[pos.Runes])
		i.Refresh()
	} else {
		i.Bell()
	}
}

func (i *Internals) Paste() {
	i.Buf = i.Buf.InsertTextAt(i.Pos, i.Cb.Text)
	i.Pos = i.Pos.Add(i.Cb.Text.Chars...)
	i.Refresh()
}

func (i *Internals) Clear() {
	mustWrite(i.Output.Write([]byte(ansi.ClearScreen)))
	i.Refresh()
}

func (i *Internals) Bell() {
	mustWrite(i.Output.Write([]byte(ansi.Bell)))
}

func (i *Internals) Refresh() {
	buf := i.Buf
	pos := i.Pos
	pos2 := utils.Position{}
	x := buf.ColLen

	for i.Prompt.ColLen+pos.Columns >= i.Cols {
		c := buf.Chars[pos2.Runes]
		pos2 = pos2.Add(c)
		pos = pos.Subtract(c)
		x -= c.ColLen
	}
	pos3 := pos2
	for pos3.Columns < buf.ColLen {
		pos3 = pos3.Add(buf.Chars[pos3.Runes])
	}
	for i.Prompt.ColLen+x >= i.Cols {
		c := buf.Chars[pos3.Runes-1]
		pos3 = pos3.Subtract(c)
		x -= c.ColLen
	}
	buf = buf.Slice(pos2, pos3)

	mustWrite(fmt.Fprintf(i.Output, "%s%s%s%s%s",
		ansi.CursorToLeftEdge,
		i.Prompt.Bytes,
		buf.Bytes,
		ansi.EraseToRight,
		fmt.Sprintf(ansi.MoveCursorForward, i.Prompt.ColLen+pos.Columns),
	))
}

func mustWrite(n int, err error) int {
	if err != nil {
		panic(err)
	}
	return n
}
