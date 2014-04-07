package utils

import "github.com/shinichy/go-wcwidth"

// char represents a character in the terminal screen
// Its size is defined as follows:
// - 1 rune
// - len(char.b) bytes
// - char.ColLen terminal columns
type Char struct {
	P      []byte
	R      rune
	ColLen int
}

func CharFromRune(r rune) Char {
	return Char{[]byte(string(r)), r, wcwidth.WcwidthUcs(r)}
}

func (c Char) Clone() Char {
	b := make([]byte, len(c.P))
	copy(b, c.P)
	c.P = b
	return c
}

// Text represents a sequence of characters
// Its size is defined as follows:
// - len(Text.Chars) runes
// - len(Text.Bytes) bytes
// - Text.ColLen terminal columns
type Text struct {
	Chars  []Char
	Bytes  []byte
	ColLen int
}

func TextFromString(s string) Text {
	t := Text{Chars: make([]Char, 0, len(s)), Bytes: make([]byte, 0, len(s))}
	for _, r := range s {
		c := CharFromRune(r)
		t.Chars = append(t.Chars, c)
		t.Bytes = append(t.Bytes, c.P...)
		t.ColLen += c.ColLen
	}
	return t
}

func (t Text) AppendChar(c Char) Text {
	return Text{append(t.Chars, c), append(t.Bytes, c.P...), t.ColLen + c.ColLen}
}

func (t Text) AppendText(n Text) Text {
	return Text{append(t.Chars, n.Chars...), append(t.Bytes, n.Bytes...), t.ColLen + n.ColLen}
}

func (t Text) InsertCharAt(pos Position, c Char) Text {
	chars := make([]Char, len(t.Chars)+1)
	copy(chars, t.Chars[:pos.Runes])
	chars[pos.Runes] = c
	copy(chars[pos.Runes+1:], t.Chars[pos.Runes:])

	bytes := make([]byte, len(t.Bytes)+len(c.P))
	copy(bytes, t.Bytes[:pos.Bytes])
	copy(bytes[pos.Bytes:], c.P)
	copy(bytes[pos.Bytes+len(c.P):], t.Bytes[pos.Bytes:])
	return Text{chars, bytes, t.ColLen + c.ColLen}
}

func (t Text) InsertTextAt(pos Position, n Text) Text {
	chars := make([]Char, len(t.Chars)+len(n.Chars))
	copy(chars, t.Chars[:pos.Runes])
	copy(chars[pos.Runes:], n.Chars)
	copy(chars[pos.Runes+len(n.Chars):], t.Chars[pos.Runes:])

	bytes := make([]byte, len(t.Bytes)+len(n.Bytes))
	copy(bytes, t.Bytes[:pos.Bytes])
	copy(bytes[pos.Bytes+len(n.Bytes):], t.Bytes[pos.Bytes:])
	copy(bytes[pos.Bytes:], n.Bytes)

	return Text{chars, bytes, t.ColLen + n.ColLen}
}

func (t Text) RemoveCharAt(pos Position) Text {
	c := t.Chars[pos.Runes]
	t.Bytes = append(t.Bytes[:pos.Bytes], t.Bytes[pos.Bytes+len(c.P):]...)
	t.Chars = append(t.Chars[:pos.Runes], t.Chars[pos.Runes+1:]...)
	t.ColLen -= c.ColLen
	return t
}

func (t Text) Slice(segment ...Position) Text {
	switch len(segment) {
	case 1:
		t.Chars = t.Chars[segment[0].Runes:]
		t.Bytes = t.Bytes[segment[0].Bytes:]
		t.ColLen -= segment[0].Columns
	case 2:
		t.Chars = t.Chars[segment[0].Runes:segment[1].Runes]
		t.Bytes = t.Bytes[segment[0].Bytes:segment[1].Bytes]
		t.ColLen = segment[1].Columns - segment[0].Columns
	default:
		panic("Slice expects 1 or 2 Position arguments")
	}
	return t
}

func (t Text) Clone() Text {
	chars := make([]Char, len(t.Chars))
	for i, c := range t.Chars {
		chars[i] = c.Clone()
	}
	t.Chars = chars
	b := make([]byte, len(t.Bytes))
	copy(b, t.Bytes)
	t.Bytes = b
	return t
}

func (t Text) String() string {
	return string(t.Bytes)
}

type Position struct {
	Bytes   int
	Runes   int
	Columns int
}

func (pos Position) Add(chars ...Char) Position {
	for _, c := range chars {
		pos.Runes++
		pos.Bytes += len(c.P)
		pos.Columns += c.ColLen
	}
	return pos
}

func (pos Position) Subtract(chars ...Char) Position {
	for _, c := range chars {
		pos.Runes--
		pos.Bytes -= len(c.P)
		pos.Columns -= c.ColLen
	}
	return pos
}

type Clipboard struct {
	Text    Text
	Partial bool
}

type History struct {
	Saved []string
	Tmp   []string
	Index int
}
