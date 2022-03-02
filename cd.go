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
	cacheName = "dotCache"
)

var (
	osStat = os.Stat
)

type Dot struct {
	// Aliases is a map from alias type to alias to absolute directory path.
	Aliases   map[string]map[string][]string
	NumRecurs int
	Caches map[string][][]string

	changed bool
}

func (d *Dot) AliasMap() map[string]map[string][]string {
	if d.Aliases == nil {
		d.Aliases = map[string]map[string][]string{}
	}
	return d.Aliases
}

func (d *Dot) Cache() map[string][][]string {
	if d.Caches == nil {
		d.Caches = map[string][][]string{}
	}
	return d.Caches
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
	if !data.Has(pathArg) {
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
	return strings.ReplaceAll(strings.ReplaceAll(path, " ", "\\ "), "\\", "\\\\")
}

func (d *Dot) Node() *command.Node {
	opts := []command.ArgOpt[string]{
		&command.Completor[string]{
			SuggestionFetcher: &command.FileFetcher[string]{
				Directory:   d.directory(),
				IgnoreFiles: true,
			},
		},
		command.NewTransformer[string](func(v string) (string, error) {
			var path []string
			for i := 0; i < d.NumRecurs; i++ {
				path = append(path, "..")
			}
			path = append(path, v)
			a, err := filepath.Abs(filepath.Join(path...))
			if err != nil {
				return "", fmt.Errorf("failed to transform file path: %v", err)
			}

			return a, nil
		}, false),
	}

	subOpts := []command.ArgOpt[[]string]{
		&command.Completor[[]string]{
			SuggestionFetcher: &subPathFetcher{d},
		},
	}

	n := command.SerialNodes(
		command.Description("Changes directories"),
		command.OptionalArg[string](pathArg, "destination directory", opts...),
		command.ListArg[string](subPathArg, "subdirectories to continue to", 0, command.UnboundedList, subOpts...),
		command.ExecutableNode(d.cd),
	)
	if d.NumRecurs == 0 {
		// Only uses cache and aliases with the single dot command.
		return command.BranchNode(
			map[string]*command.Node{
				"-": command.SerialNodes(
					command.Description("Go to the previous directory"),
					command.SimpleExecutableNode("cd -"),
				),
			},
			// TODO: prefer directory over alias
			command.CacheNode(cacheName, d, command.AliasNode(dirAliaserName, d, n)),
			command.DontCompleteSubcommands(),
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

func (spf *subPathFetcher) Fetch(sl []string, d *command.Data) (*command.Completion, error) {
	base := filepath.Join(append(
		[]string{
			spf.d.directory(),
			d.String(pathArg),
		},
		// Remove last file/directory part from provided path
		sl[:len(sl)-1]...,
	)...)

	ff := &command.FileFetcher[[]string]{
		Directory:   base,
		IgnoreFiles: true,
	}
	return ff.Fetch(sl, d)
}
