# `cd` Package

The `cd` package implements a `github.com/leep-frog/command/sourcerer.CLI` for moving around directories. It allows directory shortcuts as well as a recursive `up` flag for cd-ing in a parent directory.

Add the following to your bash profile to get the most out of the CLI:

```bash
# Load the cd command
sourcerer $GOPATH/src/cd/cmd/ cd

# Helpful common directory CLIs
# Note that autocomplete will properly work for all of these commands!
aliaser g . $GOPATH/src
aliaser gb . $GOPATH/bin
aliaser gc . $GOPATH/cmd
# etc.

# Helpful recursive CLIs
# Also available as sourcerer.Option in go with `cd.DotAliaser(2)`
aliaser .. . -u 1
aliaser ... . -u 2
aliaser .... . -u 3
aliaser ..... . -u 4
# etc.
```
