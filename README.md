Package uniline provides a simple readline API written in pure Go with
Unicode support. It allows users to interactively input and edit a line
from the terminal.

Most of the usual GNU readline capabilities and control keys are
implemented. If the provided input source is not a TTY or not an
ANSI-compatible TTY, uniline falls back to scanning each line using
`bufio.ScanLines`.

## Install

### With Go

```Go
go get github.com/tiborvass/uniline
```

## Documentation

[https://godoc.org/github.com/tiborvass/uniline](https://godoc.org/github.com/tiborvass/uniline)

## Example

```Go
package main

import (
	"fmt"
	"github.com/tiborvass/uniline"
)

func main() {
	prompt := "> "
	scanner := uniline.DefaultScanner()
	for scanner.Scan(prompt) {
		line := scanner.Text()
		if len(line) > 0 {
			scanner.AddToHistory(line)
			fmt.Println(line)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
```

## TODO

- Multiline
- History search
- Tab completion
- Catch SIGWINCH when window resizes

## License

MIT
