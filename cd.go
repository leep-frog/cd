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

func (*Dot) Load(jsn string) error    { return nil }
func (*Dot) Changed() bool            { return false }
func (*Dot) Option() *commands.Option { return nil }
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

func (d *Dot) cd(ws *commands.WorldState) {
	path := d.directory()
	if ws.Args[pathArg].Provided() {
		path = filepath.Join(path, ws.Args[pathArg].String())
	}

	if fi, err := osStat(path); err == nil && !fi.IsDir() {
		path = filepath.Dir(path)
	}

	ws.Executable = [][]string{{"cd", path}}
}

func (d *Dot) Node() *commands.Node {
	cmp := &commands.Completor{
		SuggestionFetcher: &commands.FileFetcher{
			Directory: d.directory(),
		},
	}

	return commands.SerialNodes(
		commands.StringArg(pathArg, false, cmp),
		commands.ExecutorNode(d.cd),
	)
}

func DotCLI(NumRecurs int) *Dot {
	return &Dot{
		NumRecurs: NumRecurs,
	}
}
