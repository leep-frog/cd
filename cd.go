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
	pathArg        = "PATH"
	subPathArg     = "SUB_PATH"
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

func (d *Dot) cd(output command.Output, data *command.Data) ([]string, error) {
	if !data.HasArg(pathArg) {
		return []string{fmt.Sprintf("cd %s", fp(d.directory()))}, nil
	}

	path := data.String(pathArg)
	if fi, err := osStat(path); err == nil && !fi.IsDir() {
		path = filepath.Dir(path)
	}

	subPaths := append([]string{path}, data.StringList(subPathArg)...)
	return []string{fmt.Sprintf("cd %s", fp(filepath.Join(subPaths...)))}, nil
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
			path = append(path, v.ToString())
			a, err := filepath.Abs(filepath.Join(path...))
			if err != nil {
				return nil, fmt.Errorf("failed to transform file path: %v", err)
			}

			return command.StringValue(a), nil
		}, false),
	}

	subOpts := []command.ArgOpt{
		&command.Completor{
			SuggestionFetcher: &subPathFetcher{d},
		},
	}

	n := command.SerialNodes(
		command.Description("Changes directories"),
		//command.OptionalStringNode(pathArg, "destination directory", opts...),
		//command.StringListNode(pathArg, "destination directory", 0, command.UnboundedList, opts...),
		command.OptionalStringNode(pathArg, "destination directory", opts...),
		command.StringListNode(subPathArg, "subdirectories to continue to", 0, command.UnboundedList, subOpts...),
		command.ExecutableNode(d.cd),
	)
	if d.NumRecurs == 0 {
		// Only uses aliases with the single dot command.
		return command.BranchNode(
			map[string]*command.Node{
				"-": command.SerialNodes(
					command.Description("Go to the previous directory"),
					command.SimpleExecutableNode("cd -"),
				),
			},
			// TODO: prefer directory over alias
			command.AliasNode(dirAliaserName, d, n),
			false,
		)
	}
	return n
}

func DotCLI(NumRecurs int) *Dot {
	return &Dot{
		NumRecurs: NumRecurs,
	}
}

type subPathFetcher struct {
	d *Dot
}

func (spf *subPathFetcher) Fetch(v *command.Value, d *command.Data) (*command.Completion, error) {
	sl := v.ToStringList()
	sl = sl[:len(sl)-1]

	base := filepath.Join(append(
		[]string{
			spf.d.directory(),
			d.String(pathArg),
		},
		sl...,
	)...)

	ff := &command.FileFetcher{
		Directory:   base,
		IgnoreFiles: true,
	}
	return ff.Fetch(v, d)
}
