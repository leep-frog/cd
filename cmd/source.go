package main

import (
	"os"

	"github.com/leep-frog/cd"
	"github.com/leep-frog/command/sourcerer"
)

func main() {
	os.Exit(sourcerer.Source("cdCLI", []sourcerer.CLI{cd.DotCLI()}))
	/*os.Exit(sourcerer.Source(
		[]sourcerer.CLI{cd.DotCLI()},
		cd.MinusAliaser(),
		cd.DotAliasersUpTo(10),
	))*/
}
