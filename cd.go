package cd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
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

	upFlag = command.Flag[int]("up", 'u', "Number of directories to go up when cd-ing", command.Default(0), command.NonNegative[int]())
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
func (d *Dot) Name() string  { return "." }

func getDirectory(data *command.Data, extra ...string) string {
	upTo := upFlag.Get(data)
	path := make([]string, upTo)
	for i := 0; i < upTo; i++ {
		path[i] = ".."
	}
	path = append(path, extra...)
	return filepath.Join(path...)
}

var (
	newShell = cache.NewShell
)

func (d *Dot) getHistory() (*cache.Cache, *History, error) {
	c, err := newShell()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get shell cache: %v", err)
	}

	h := &History{}
	if _, err := c.GetStruct(shellCacheKey, h); err != nil {
		return nil, nil, fmt.Errorf("failed to get struct data: %v", err)
	}

	return c, h, nil
}

func (d *Dot) updateHistory(output command.Output, data *command.Data) error {
	// Get the cache data
	c, h, err := d.getHistory()
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
	dir := command.GetwdFromData(data)

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

func relativeFetcher() command.Completer[string] {
	return command.CompleterFromFunc(func(s string, data *command.Data) (*command.Completion, error) {
		f := &command.FileCompleter[string]{
			Directory:   getDirectory(data),
			IgnoreFiles: true,
		}
		return f.Complete(s, data)
	})
}

type relativeTransformer struct {
}

func (d *Dot) Node() *command.Node {
	opts := []command.ArgOpt[string]{
		relativeFetcher(),
		command.CompleteForExecute[string](command.CompleteForExecuteBestEffort()),
		&command.Transformer[string]{F: func(v string, data *command.Data) (string, error) {
			return filepath.Abs(getDirectory(data, v))
		}},
	}

	subOpts := []command.ArgOpt[[]string]{
		command.CompleteForExecute[[]string](command.CompleteForExecuteBestEffort()),
		subPathFetcher(),
	}

	dfltNode := command.ShortcutNode(dirShortcutName, d, command.SerialNodes(
		command.Description("Changes directories"),
		command.EchoExecuteData(),
		command.FlagNode(
			upFlag,
		),
		command.OptionalArg(pathArg, "destination directory", opts...),
		command.ListArg(subPathArg, "subdirectories to continue to", 0, command.UnboundedList, subOpts...),
		command.Getwd(),
		command.ExecutableNode(d.cd),
		&command.ExecutorProcessor{F: d.updateHistory},
	))

	return command.AsNode(&command.BranchNode{
		Branches: map[string]*command.Node{
			"hist": command.SerialNodes(
				&command.ExecutorProcessor{F: func(o command.Output, data *command.Data) error {
					c, h, err := d.getHistory()
					if err != nil {
						return o.Err(err)
					}
					o.Stdoutln("WD: ", command.GetwdFromData(data))
					o.Stdoutln("HISTORY: ", h)
					o.Stdoutln("CACHE: ", c.Dir, c)
					return nil
				}},
			),
			"-": command.SerialNodes(
				command.Description("Go to the previous directory"),
				command.Getwd(),
				command.ExecutableNode(func(output command.Output, data *command.Data) ([]string, error) {
					c, h, err := d.getHistory()
					if err != nil {
						return nil, output.Err(err)
					}
					wd := command.GetwdFromData(data)
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
	})
}

func DotCLI() *Dot {
	return &Dot{}
}

// MinusAliaser returns an alias for ". -"
func MinusAliaser() sourcerer.Option {
	return sourcerer.NewAliaser("-", ".", "-")
}

// DotAliaser returns an aliaser option that searches `n` directories up with
// an alias of n `.` characters.
func DotAliaser(n int) sourcerer.Option {
	return sourcerer.NewAliaser(strings.Repeat(".", n), fmt.Sprintf(". -u %d", n-1))
}

// DotAliasersUpTo returns `DotAliaser` options from 2 to `n`.
func DotAliasersUpTo(n int) sourcerer.Option {
	m := map[string][]string{}
	for i := 2; i <= n; i++ {
		m[strings.Repeat(".", i)] = []string{".", "-u", fmt.Sprintf("%d", i-1)}
	}
	return sourcerer.Aliasers(m)
}

func subPathFetcher() command.Completer[[]string] {
	return command.CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
		base := filepath.Join(append(
			[]string{getDirectory(d, d.String(pathArg))},
			// Remove last file/directory part from provided path
			sl[:len(sl)-1]...,
		)...)

		ff := &command.FileCompleter[[]string]{
			Directory:   base,
			IgnoreFiles: true,
		}
		return ff.Complete(sl, d)
	})
}
