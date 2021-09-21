package cd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command"
)

const (
	pathArg        = "path"
	dirAliaserName = "dirAliases"
)

var (
	osStat = os.Stat
)

type Dot struct {
	// Aliases is a map from alias type to alias to absolute directory path.
	Aliases   map[string]map[string][]string
	NumRecurs int

	changed bool
}

func (d *Dot) AliasMap() map[string]map[string][]string {
	if d.Aliases == nil {
		d.Aliases = map[string]map[string][]string{}
	}
	return d.Aliases
}

// Load creates a dot object from a JSON staring.
func (d *Dot) Load(jsn string) error {
	if d != nil {
		r := d.NumRecurs
		defer func() { d.NumRecurs = r }()
	}

	if jsn == "" {
		d = &Dot{}
		return nil
	}

	if err := json.Unmarshal([]byte(jsn), d); err != nil {
		return fmt.Errorf("failed to unmarshal dot json: %v", err)
	}
	return nil
}

func (d *Dot) Changed() bool { return d.changed }
func (d *Dot) MarkChanged()  { d.changed = true }
func (*Dot) Setup() []string { return nil }
func (d *Dot) Name() string {
	return strings.Repeat(".", d.NumRecurs+1)
}

func (d *Dot) directory() string {
	path := make([]string, d.NumRecurs)
	for i := 0; i < d.NumRecurs; i++ {
		path[i] = ".."
	}
	return filepath.Join(path...)
}

func (d *Dot) cd(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	if !data.HasArg(pathArg) {
		eData.Executable = append(eData.Executable, fmt.Sprintf("cd %s", fp(d.directory())))
		return nil
	}

	path := data.String(pathArg)
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
	opts := []command.ArgOpt{
		&command.Completor{
			SuggestionFetcher: &command.FileFetcher{
				Directory:   d.directory(),
				IgnoreFiles: true,
			},
		},
		command.Transformer(command.StringType, func(v *command.Value) (*command.Value, error) {
			var path []string
			for i := 0; i < d.NumRecurs; i++ {
				path = append(path, "..")
			}
			path = append(path, v.String())
			a, err := filepath.Abs(filepath.Join(path...))
			if err != nil {
				return nil, fmt.Errorf("failed to transform file path: %v", err)
			}
			return command.StringValue(a), nil
		}, false),
	}

	return command.AliasNode(dirAliaserName, d, command.SerialNodes(
		command.OptionalStringNode(pathArg, opts...),
		command.SimpleProcessor(d.cd, nil),
	))
}

func DotCLI(NumRecurs int) *Dot {
	return &Dot{
		NumRecurs: NumRecurs,
	}
}
