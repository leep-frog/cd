package cd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/commands/commands"
)

const (
	pathArg = "path"
)

var (
	osStat = os.Stat
)

type Dot struct {
	NumRecurs int
}

func (*Dot) Load(jsn string) error { return nil }
func (*Dot) Changed() bool         { return false }
func (d *Dot) Name() string {
	return fmt.Sprintf("%d-dir-dot", d.NumRecurs)
}
func (d *Dot) Alias() string {
	return strings.Repeat(".", d.NumRecurs+1)
}

func (d *Dot) directory() string {
	path := make([]string, d.NumRecurs)
	for i := range path {
		path[i] = ".."
	}
	return filepath.Join(path...)
}

func (d *Dot) cd(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo) (*commands.ExecutorResponse, bool) {
	path := d.directory()
	if strPtr := args[pathArg].String(); strPtr != nil {
		path = filepath.Join(path, *strPtr)
	}

	if fi, err := osStat(path); err == nil && !fi.IsDir() {
		path = filepath.Dir(path)
	}

	return &commands.ExecutorResponse{
		Executable: []string{"cd", path},
	}, true
}

func (d *Dot) Command() commands.Command {
	cmp := &commands.Completor{
		SuggestionFetcher: &commands.FileFetcher{
			Directory: d.directory(),
		},
	}

	return &commands.TerminusCommand{
		Executor: d.cd,
		Args: []commands.Arg{
			commands.StringArg(pathArg, false, cmp),
		},
	}
}

func DotCLI(NumRecurs int) *Dot {
	return &Dot{
		NumRecurs: NumRecurs,
	}
}
