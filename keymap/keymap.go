package keymap

import (
	"github.com/tiborvass/uniline/ansi"
	"github.com/tiborvass/uniline/internals"
)

// Keymap is a hash table mapping Ansi codes to functions.
// There are no locks on this map, if accessing it concurrently
// please consider it as a static map (1 initial write, THEN as many concurrent reads as you wish)
type Keymap map[ansi.Code]func(*internals.Internals)

// DefaultKeymap returns a copy of the default Keymap
// Useful if inspection/customization is needed.
func DefaultKeymap() Keymap {
	return Keymap{
		ansi.NEWLINE:         (*internals.Internals).Enter,
		ansi.CARRIAGE_RETURN: (*internals.Internals).Enter,
		ansi.CTRL_C:          (*internals.Internals).Interrupt,
		ansi.CTRL_D:          (*internals.Internals).DeleteOrEOF,
		ansi.CTRL_H:          (*internals.Internals).Backspace,
		ansi.BACKSPACE:       (*internals.Internals).Backspace,
		ansi.CTRL_L:          (*internals.Internals).Clear,
		ansi.CTRL_T:          (*internals.Internals).SwapChars,

		ansi.CTRL_B: (*internals.Internals).MoveLeft,
		ansi.CTRL_F: (*internals.Internals).MoveRight,
		ansi.CTRL_P: (*internals.Internals).HistoryBack,
		ansi.CTRL_N: (*internals.Internals).HistoryForward,

		ansi.CTRL_U: (*internals.Internals).CutLineLeft,
		ansi.CTRL_K: (*internals.Internals).CutLineRight,
		ansi.CTRL_A: (*internals.Internals).MoveBeginning,
		ansi.CTRL_E: (*internals.Internals).MoveEnd,
		ansi.CTRL_W: (*internals.Internals).CutPrevWord,
		ansi.CTRL_Y: (*internals.Internals).Paste,

		// Escape sequences
		ansi.START_ESCAPE_SEQ: nil,

		ansi.META_B:     (*internals.Internals).MoveWordLeft,
		ansi.META_LEFT:  (*internals.Internals).MoveWordLeft,
		ansi.META_F:     (*internals.Internals).MoveWordRight,
		ansi.META_RIGHT: (*internals.Internals).MoveWordRight,

		ansi.LEFT:  (*internals.Internals).MoveLeft,
		ansi.RIGHT: (*internals.Internals).MoveRight,
		ansi.UP:    (*internals.Internals).HistoryBack,
		ansi.DOWN:  (*internals.Internals).HistoryForward,

		// Extended escape
		ansi.START_EXTENDED_ESCAPE_SEQ:   nil,
		ansi.START_EXTENDED_ESCAPE_SEQ_3: nil,

		ansi.DELETE: (*internals.Internals).Delete, // Delete key
	}
}
