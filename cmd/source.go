package main

import (
	"github.com/leep-frog/cd"
	"github.com/leep-frog/command/sourcerer"
)

func main() {
	var clis []sourcerer.CLI
	for i := 0; i < 10; i++ {
		clis = append(clis, cd.DotCLI(i))
	}
	sourcerer.Source(clis...)
}