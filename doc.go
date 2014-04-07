/*
 Author: Tibor Vass (gh: @tiborvass)
 License: MIT
*/

/*
Package uniline provides a simple readline API written in pure Go with Unicode support.
It allows users to interactively input and edit a line from the terminal.

Most of the usual GNU readline capabilities and control keys are implemented.
If the provided input source is not a TTY or not an ANSI-compatible TTY, uniline falls back to scanning each line using bufio.ScanLines.

TODO:

- add support for multiline

- add support for History search

- add support for tab completion

- catch SIGWINCH and adjust cols
*/
package uniline
