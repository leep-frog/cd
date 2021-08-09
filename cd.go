package cd

import (
	"fmt"
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
	path := make([]string, d.NumRecurs+1)
	path[0] = "."
	for i := 1; i <= d.NumRecurs; i++ {
		path[i] = ".."
	}
	return filepath.Join(path...)
}

func (d *Dot) cd(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	path := d.directory()
	if data.Provided(pathArg) {
		path = filepath.Join(path, data.String(pathArg))
	}

	if fi, err := osStat(path); err == nil && !fi.IsDir() {
		path = filepath.Dir(path)
	}

	eData.Executable = append(eData.Executable, fmt.Sprintf("cd %s", fp(path)))
	return nil
}

func fp(path string) string {
	// Needed for use in msys2 mingw.
	return strings.ReplaceAll(path, "\\", "\\\\")
}

func (d *Dot) Node() *command.Node {
	ao := &command.ArgOpt{
		Completor: &command.Completor{
			SuggestionFetcher: &command.FileFetcher{
				Directory:   d.directory(),
				IgnoreFiles: true,
			},
		},
	}

	return command.SerialNodes(
		command.OptionalStringNode(pathArg, ao),
		command.SimpleProcessor(d.cd, nil),
	)
}

func DotCLI(NumRecurs int) *Dot {
	return &Dot{
		NumRecurs: NumRecurs,
	}
}
