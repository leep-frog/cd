package cd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command/cache"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/sourcerer"
)

const (
	pathArg         = "PATH"
	subPathArg      = "SUB_PATH"
	dirShortcutName = "dirShortcuts"
	shellCacheKey   = "leep-cd-shell"
)

var (
	osStat = os.Stat

	upFlag       = commander.Flag[int]("up", 'u', "Number of directories to go up when cd-ing", commander.Default(0), commander.NonNegative[int]())
	parentDirArg = commander.Arg[string]("PARENT_DIR", "Name of the parent directory to go up to",
		&commander.Complexecute[string]{Lenient: true},
		commander.CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
			var r []string
			prev := commander.Getwd.Get(d)
			for pwd := filepath.Dir(prev); pwd != prev; prev, pwd = pwd, filepath.Dir(pwd) {
				base := filepath.Base(pwd)
				if base != `/` && base != `\` {
					r = append(r, base)
				}
			}

			return &command.Completion{
				CaseInsensitive: true,
				Suggestions:     r,
			}, nil
		}),
	)
)

type Dot struct {
	// Shortcuts is a map from shortcut type to shortcuts to absolute directory path.
	Shortcuts map[string]map[string][]string

	changed bool
}

func (d *Dot) ShortcutMap() map[string]map[string][]string {
	if d.Shortcuts == nil {
		d.Shortcuts = map[string]map[string][]string{}
	}
	return d.Shortcuts
}

func (d *Dot) Changed() bool { return d.changed }
func (d *Dot) MarkChanged()  { d.changed = true }
func (*Dot) Setup() []string { return nil }
func (d *Dot) Name() string  { return dotName }

func getDirectory(data *command.Data, extra ...string) string {
	upTo := upFlag.Get(data)
	path := make([]string, upTo)
	for i := 0; i < upTo; i++ {
		path[i] = ".."
	}
	path = append(path, extra...)
	return filepath.Join(path...)
}

func (d *Dot) getHistory(data *command.Data) (*cache.Cache, *History, error) {
	c := cache.ShellFromData(data)

	h := &History{}
	if _, err := c.GetStruct(shellCacheKey, h); err != nil {
		return nil, nil, fmt.Errorf("failed to get struct data: %v", err)
	}

	return c, h, nil
}

func (d *Dot) updateHistory(output command.Output, data *command.Data) error {
	// Get the cache data
	c, h, err := d.getHistory(data)
	if err != nil {
		return output.Err(err)
	}

	// Update the cache data
	return output.Err(h.append(c, data))
}

type History struct {
	PrevDirs []string
}

func (h *History) append(c *cache.Cache, data *command.Data) error {
	dir := commander.Getwd.Get(data)

	// No need to update if previous directory is the same.
	if len(h.PrevDirs) > 0 && h.PrevDirs[len(h.PrevDirs)-1] == dir {
		return nil
	}

	// Otherwise append, truncate, and save.
	h.PrevDirs = append(h.PrevDirs, dir)
	if len(h.PrevDirs) > 2 {
		h.PrevDirs = h.PrevDirs[len(h.PrevDirs)-2:]
	}
	if err := c.PutStruct(shellCacheKey, h); err != nil {
		return fmt.Errorf("failed to save history: %v", err)
	}
	return nil
}

func (d *Dot) cd(output command.Output, data *command.Data) ([]string, error) {
	if !data.Has(pathArg) {
		if dir := getDirectory(data); dir != "" {
			return []string{fmt.Sprintf("cd %q", dir)}, nil
		}
		return []string{"cd"}, nil
	}

	path := data.String(pathArg)
	if fi, err := osStat(path); err == nil && !fi.IsDir() {
		path = filepath.Dir(path)
	}

	subPaths := append([]string{path}, data.StringList(subPathArg)...)
	return []string{fmt.Sprintf("cd %q", filepath.Join(subPaths...))}, nil
}

func relativeFetcher() commander.Completer[string] {
	return commander.CompleterFromFunc(func(s string, data *command.Data) (*command.Completion, error) {
		f := &commander.FileCompleter[string]{
			Directory:   getDirectory(data),
			IgnoreFiles: true,
			ExcludePwd:  true,
		}
		return f.Complete(s, data)
	})
}

type relativeTransformer struct {
}

func (d *Dot) Node() command.Node {
	opts := []commander.ArgumentOption[string]{
		relativeFetcher(),
		&commander.Complexecute[string]{Lenient: true},
		&commander.Transformer[string]{F: func(v string, data *command.Data) (string, error) {
			return filepath.Abs(getDirectory(data, v))
		}},
	}

	subOpts := []commander.ArgumentOption[[]string]{
		&commander.Complexecute[[]string]{Lenient: true},
		subPathFetcher(),
	}

	dfltNode := commander.ShortcutNode(dirShortcutName, d, commander.SerialNodes(
		commander.Description("Changes directories"),
		commander.EchoExecuteData(),
		cache.ShellProcessor(),
		commander.FlagProcessor(
			upFlag,
		),
		commander.OptionalArg(pathArg, "destination directory", opts...),
		commander.ListArg(subPathArg, "subdirectories to continue to", 0, command.UnboundedList, subOpts...),
		commander.Getwd,
		commander.ExecutableProcessor(d.cd),
		&commander.ExecutorProcessor{F: d.updateHistory},
	))

	return &commander.BranchNode{
		Branches: map[string]command.Node{
			"parent": commander.SerialNodes(
				commander.Getwd,
				parentDirArg,
				cache.ShellProcessor(),
				commander.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					dir := parentDirArg.Get(d)
					prev := commander.Getwd.Get(d)
					for pwd := filepath.Dir(prev); pwd != prev; prev, pwd = pwd, filepath.Dir(pwd) {
						if filepath.Base(pwd) == dir {
							return []string{
								fmt.Sprintf("cd %q", pwd),
							}, nil
						}
					}
					return nil, o.Stderrf("%s must be a parent directory\n", parentDirArg.Name())
				}),
				&commander.ExecutorProcessor{F: d.updateHistory},
			),
			"hist": commander.SerialNodes(
				cache.ShellProcessor(),
				&commander.ExecutorProcessor{F: func(o command.Output, data *command.Data) error {
					c, h, err := d.getHistory(data)
					if err != nil {
						return o.Err(err)
					}
					o.Stdoutln("WD: ", commander.Getwd.Get(data))
					o.Stdoutln("HISTORY: ", h)
					o.Stdoutln("CACHE: ", c.Dir, c)
					return nil
				}},
			),
			"-": commander.SerialNodes(
				commander.Description("Go to the previous directory"),
				commander.Getwd,
				cache.ShellProcessor(),
				commander.ExecutableProcessor(func(output command.Output, data *command.Data) ([]string, error) {
					c, h, err := d.getHistory(data)
					if err != nil {
						return nil, output.Err(err)
					}
					wd := commander.Getwd.Get(data)
					pd := wd
					for i := len(h.PrevDirs) - 1; pd == wd && i >= 0; i-- {
						pd = h.PrevDirs[i]
					}
					cmd := "cd"
					if pd != wd {
						cmd = fmt.Sprintf("cd %q", pd)
					}

					return []string{cmd}, output.Err(h.append(c, data))
				}),
			),
		},
		Default:           dfltNode,
		DefaultCompletion: true,
	}
}

func DotCLI() *Dot {
	return &Dot{}
}

var (
	dotName = "d"
)

// MinusAliaser returns an alias for ". -"
func MinusAliaser() sourcerer.Option {
	return sourcerer.NewAliaser("m", dotName, "-")
}

// ParentAliaser returns an alias for "d parent"
func ParentAliaser() sourcerer.Option {
	return sourcerer.NewAliaser("up", dotName, "parent")
}

// DotAliaser returns an `Aliaser` option that searches `n` directories up with
// an alias of n `.` characters.
func DotAliaser(n int) sourcerer.Option {
	return sourcerer.NewAliaser(strings.Repeat(dotName, n), fmt.Sprintf(". -u %d", n-1))
}

// DotAliasersUpTo returns `DotAliaser` options from 2 to `n`.
func DotAliasersUpTo(prefix, suffix string, n int) sourcerer.Option {
	m := map[string][]string{}
	for i := 1; i <= n; i++ {
		m[fmt.Sprintf("%s%s", prefix, strings.Repeat(suffix, i))] = []string{dotName, "-u", fmt.Sprintf("%d", i)}
		m[fmt.Sprintf("%s%d", prefix, i)] = []string{dotName, "-u", fmt.Sprintf("%d", i)}
	}
	return sourcerer.Aliasers(m)
}

func subPathFetcher() commander.Completer[[]string] {
	return commander.CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
		base := filepath.Join(append(
			[]string{getDirectory(d, d.String(pathArg))},
			// Remove last file/directory part from provided path
			sl[:len(sl)-1]...,
		)...)

		ff := &commander.FileCompleter[[]string]{
			Directory:   base,
			IgnoreFiles: true,
			ExcludePwd:  true,
		}
		return ff.Complete(sl, d)
	})
}
