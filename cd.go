package cd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command"
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
func (*Dot) Setup() []string       { return nil }
func (d *Dot) Name() string {
	return strings.Repeat(".", d.NumRecurs+1)
}

func (d *Dot) directory() string {
	path := make([]string, d.NumRecurs)
	for i := range path {
		path[i] = ".."
	}
	return filepath.Join(path...)
}

func (d *Dot) cd(output command.Output, data *command.Data) error {
	path := d.directory()
	if data.Values[pathArg].Provided() {
		path = filepath.Join(path, data.Values[pathArg].String())
	}

	if fi, err := osStat(path); err == nil && !fi.IsDir() {
		path = filepath.Dir(path)
	}

	return os.Chdir(path)
}

func (d *Dot) Node() *command.Node {
	ao := &command.ArgOpt{
		Completor: &command.Completor{
			SuggestionFetcher: &command.FileFetcher{
				Directory: d.directory(),
			},
		},
	}

	return command.SerialNodes(
		command.OptionalStringNode(pathArg, ao),
		command.ExecutorNode(d.cd),
	)
}

func DotCLI(NumRecurs int) *Dot {
	return &Dot{
		NumRecurs: NumRecurs,
	}
}
