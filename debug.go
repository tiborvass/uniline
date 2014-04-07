package uniline

import (
	"fmt"
	"os"
)

var debugFile *os.File

func init() {
	name := os.Getenv("DEBUG_UNILINE")
	fmt.Println(name)
	f, err := os.Create(name)
	if err == nil {
		debugFile = f
		debug("init %s", name)
	}
}

func debug(format string, args ...interface{}) {
	if debugFile != nil {
		fmt.Fprintf(debugFile, format+"\n", args...)
	}
}
