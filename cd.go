package cd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

const (
	pathArg         = "PATH"
	subPathArg      = "SUB_PATH"
	dirShortcutName = "dirShortcuts"
	cacheName       = "dotCache"
)

var (
	osStat = os.Stat
)

type Dot struct {
	// Shortcuts is a map from shortcut type to shortcuts to absolute directory path.
	Shortcuts map[string]map[string][]string
	Caches    map[string][][]string

	changed bool
}

func (d *Dot) ShortcutMap() map[string]map[string][]string {
	if d.Shortcuts == nil {
		d.Shortcuts = map[string]map[string][]string{}
	}
	return d.Shortcuts
}

func (d *Dot) Cache() map[string][][]string {
	if d.Caches == nil {
		d.Caches = map[string][][]string{}
	}
	return d.Caches
}

func (d *Dot) Changed() bool { return d.changed }
func (d *Dot) MarkChanged()  { d.changed = true }
func (*Dot) Setup() []string { return nil }
func (d *Dot) Name() string  { return "." }

func getDirectory(data *command.Data, extra ...string) string {
	upTo := data.Int("up")
	path := make([]string, upTo)
	for i := 0; i < upTo; i++ {
		path[i] = ".."
	}
	path = append(path, extra...)
	return filepath.Join(path...)
}

func (d *Dot) cd(output command.Output, data *command.Data) ([]string, error) {
	if !data.Has(pathArg) {
		return []string{fmt.Sprintf("cd %s", fp(getDirectory(data)))}, nil
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

type relativeFetcher struct{}

func (*relativeFetcher) Fetch(s string, data *command.Data) (*command.Completion, error) {
	f := &command.FileFetcher[string]{
		Directory:   getDirectory(data),
		IgnoreFiles: true,
	}
	return f.Fetch(s, data)
}

type relativeTransformer struct {
}

func (d *Dot) Node() *command.Node {
	opts := []command.ArgOpt[string]{
		&command.Completor[string]{
			Fetcher: &relativeFetcher{},
		},
		command.NewTransformer(func(v string, data *command.Data) (string, error) {
			a, err := filepath.Abs(getDirectory(data, v))
			if err != nil {
				return "", fmt.Errorf("failed to transform file path: %v", err)
			}

			return a, nil
		}, false),
	}

	subOpts := []command.ArgOpt[[]string]{
		&command.Completor[[]string]{
			Fetcher: &subPathFetcher{d},
		},
	}

	dfltNode := command.CacheNode(cacheName, d, command.ShortcutNode(dirShortcutName, d, command.SerialNodes(
		command.Description("Changes directories"),
		command.NewFlagNode(
			command.NewFlag("up", 'u', "Number of directories to go up when cd-ing", command.Default(0)),
		),
		command.OptionalArg(pathArg, "destination directory", opts...),
		command.ListArg(subPathArg, "subdirectories to continue to", 0, command.UnboundedList, subOpts...),
		command.ExecutableNode(d.cd),
	)))

	return command.BranchNode(
		map[string]*command.Node{
			"-": command.SerialNodes(
				command.Description("Go to the previous directory"),
				command.SimpleExecutableNode("cd -"),
			),
		},
		dfltNode,
		command.DontCompleteSubcommands(),
	)
}

func DotCLI() *Dot {
	return &Dot{}
}

// DotAliaser returns an aliaser option that searches `n` directories up with
// an alias of n `.` characters.
func DotAliaser(n int) sourcerer.Option {
	return sourcerer.Aliaser(strings.Repeat(".", n), fmt.Sprintf(". -u %d", n-1))
}

// DotAliasersUpTo returns `DotAliaser` options from 2 to `n`.
func DotAliasersUpTo(n int) sourcerer.Option {
	m := map[string][]string{}
	for i := 2; i <= n; i++ {
		m[strings.Repeat(".", i)] = []string{".", "-u", fmt.Sprintf("%d", i-1)}
	}
	return sourcerer.Aliasers(m)
}

type subPathFetcher struct {
	d *Dot
}

func (spf *subPathFetcher) Fetch(sl []string, d *command.Data) (*command.Completion, error) {
	base := filepath.Join(append(
		[]string{getDirectory(d, d.String(pathArg))},
		// Remove last file/directory part from provided path
		sl[:len(sl)-1]...,
	)...)

	ff := &command.FileFetcher[[]string]{
		Directory:   base,
		IgnoreFiles: true,
	}
	return ff.Fetch(sl, d)
}
