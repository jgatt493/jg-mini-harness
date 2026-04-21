package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintf(os.Stderr, "Usage: harness run <test-dir> [flags]\n")
		os.Exit(1)
	}
	fmt.Println("harness: not yet implemented")
}
