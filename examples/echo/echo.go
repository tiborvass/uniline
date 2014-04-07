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
