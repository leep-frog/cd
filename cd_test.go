package cd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/cache"
	"github.com/leep-frog/command/cache/cachetest"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
)

func filepathAbs(t *testing.T, path string) string {
	t.Helper()
	a, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Failed to get absolute file path: %v", err)
	}
	return a
}

func TestLoad(t *testing.T) {
	for _, test := range []struct {
		name string
		json string
	}{
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
		},
		{
			name: "ignores NumRecurs",
			json: `{"NumRecurs": 6}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := DotCLI()
			if err := json.Unmarshal([]byte(test.json), d); err != nil {
				t.Fatalf("UnmarshalJSON(%v) should return nil; got %v", test.json, err)
			}
		})
	}
}

type fakeFileInfo struct {
	isDir bool
}

func (*fakeFileInfo) Name() string       { return "" }
func (*fakeFileInfo) Size() int64        { return 0 }
func (*fakeFileInfo) Mode() os.FileMode  { return 0 }
func (*fakeFileInfo) ModTime() time.Time { return time.Now() }
func (ffi *fakeFileInfo) IsDir() bool    { return ffi.isDir }
func (*fakeFileInfo) Sys() interface{}   { return nil }

var (
	fileType = &fakeFileInfo{}
	dirType  = &fakeFileInfo{true}
)

func TestExecute(t *testing.T) {
	cwd := "prev/dir/1"
	wdHist := &History{[]string{cwd}}

	commandtest.StubValue(t, &dotName, ".")

	for _, test := range []struct {
		name               string
		d                  *Dot
		want               *Dot
		etc                *commandtest.ExecuteTestCase
		osStatFI           os.FileInfo
		osStatErr          error
		shellCache         *cache.Cache
		ignoreHistoryCheck bool
		wantHistory        *History
		cwdOverride        string
		noShellDataKey     bool
	}{
		{
			name:        "handles nil arguments",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():      0,
						commander.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:     "error if GetStruct error",
			osStatFI: dirType,
			d:        DotCLI(),
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: "} invalid json {",
			}),
			ignoreHistoryCheck: true,
			wantHistory:        &History{},
			etc: &commandtest.ExecuteTestCase{
				WantErr:    fmt.Errorf("failed to get struct data: failed to unmarshal cache data: invalid character '}' looking for beginning of value"),
				WantStderr: "failed to get struct data: failed to unmarshal cache data: invalid character '}' looking for beginning of value\n",
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():      0,
						commander.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "complete for execute",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: &History{PrevDirs: []string{filepathAbs(t, ".")}},
			cwdOverride: filepathAbs(t, "."),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"c"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "cmd"))},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():      0,
						"PATH":             filepathAbs(t, "cmd"),
						commander.GetwdKey: filepathAbs(t, "."),
					},
				},
			},
		},
		{
			name:        "handles basic dot",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():      0,
						commander.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles empty arguments",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():      0,
						commander.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles directory with spaces arguments",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"some thing"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, "some thing")),
					},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						pathArg:            filepathAbs(t, "some thing"),
						upFlag.Name():      0,
						commander.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles -u flag",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"-u", "2"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepath.Join("..", "..")),
					},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():      2,
						commander.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles absolute path",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{filepathAbs(t, "../../..")},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("..", "..", ".."))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:            filepathAbs(t, filepath.Join("..", "..", "..")),
					upFlag.Name():      0,
					commander.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "cds into directory of a file",
			osStatFI:    fileType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"something/somewhere.txt", "--up", "3"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("..", "..", "..", "something"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:            filepathAbs(t, filepath.Join("..", "..", "..", "something", "somewhere.txt")),
					upFlag.Name():      3,
					commander.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "cds into directory with spaces",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"some where/"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("some where"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:            filepathAbs(t, filepath.Join("some where")),
					upFlag.Name():      0,
					commander.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "0-dot cds down multiple paths",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"some", "thing", "some", "where"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("some", "thing", "some", "where"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:            filepathAbs(t, filepath.Join("some")),
					subPathArg:         []string{"thing", "some", "where"},
					upFlag.Name():      0,
					commander.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "1-dot cds down multiple paths",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"some", "thing", "-u", "1", "some", "where"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("..", "some", "thing", "some", "where"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:            filepathAbs(t, filepath.Join("..", "some")),
					subPathArg:         []string{"thing", "some", "where"},
					upFlag.Name():      1,
					commander.GetwdKey: cwd,
				}},
			},
		},
		// Minus tests
		{
			name: "minus goes to the previous directory",
			d:    DotCLI(),
			wantHistory: &History{[]string{
				"old/dir",
				cwd,
			}},
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir",
					},
				},
			}),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "minus goes home if no history",
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
				}},
			},
		},
		// History tests
		{
			name:     "dot history gets truncated",
			d:        DotCLI(),
			osStatFI: dirType,
			wantHistory: &History{[]string{
				"old/dir/5",
				cwd,
			}},
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						"old/dir/2",
						"old/dir/3",
						"old/dir/4",
						"old/dir/5",
					},
				},
			}),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"somewhere"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "somewhere"))},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
					"PATH":             filepathAbs(t, "somewhere"),
					upFlag.Name():      0,
				}},
			},
		},
		{
			name: "minus history gets truncated",
			d:    DotCLI(),
			wantHistory: &History{[]string{
				"old/dir/5",
				cwd,
			}},
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						"old/dir/2",
						"old/dir/3",
						"old/dir/4",
						"old/dir/5",
					},
				},
			}),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir/5"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
				}},
			},
		},
		{
			name:     "dot history skips current directory",
			d:        DotCLI(),
			osStatFI: dirType,
			wantHistory: &History{[]string{
				"old/dir/1",
				cwd,
				"old/dir/2",
				cwd,
				cwd,
				cwd,
			}},
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						cwd,
						"old/dir/2",
						cwd,
						cwd,
						cwd,
					},
				},
			}),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"somewhere"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "somewhere"))},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
					"PATH":             filepathAbs(t, "somewhere"),
					upFlag.Name():      0,
				}},
			},
		},
		{
			name: "minus history skips current directory",
			d:    DotCLI(),
			wantHistory: &History{[]string{
				"old/dir/1",
				cwd,
				"old/dir/2",
				cwd,
				cwd,
				cwd,
			}},
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						cwd,
						"old/dir/2",
						cwd,
						cwd,
						cwd,
					},
				},
			}),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir/2"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
				}},
			},
		},
		{
			name: "dot history doesn't change if in working dir",
			d:    DotCLI(),
			wantHistory: &History{[]string{
				"old/dir/1",
				cwd,
			}},
			osStatFI: dirType,
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						cwd,
					},
				},
			}),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"somewhere"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "somewhere"))},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
					"PATH":             filepathAbs(t, "somewhere"),
					upFlag.Name():      0,
				}},
			},
		},
		{
			name: "minus history doesn't change if in working dir",
			d:    DotCLI(),
			wantHistory: &History{[]string{
				"old/dir/1",
				cwd,
			}},
			shellCache: cachetest.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						cwd,
					},
				},
			}),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir/1"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: cwd,
				}},
			},
		},
		// parent tests
		{
			name:           "parent fails if no arg",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    "/abc/def/ghi",
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"parent"},
				WantErr:    fmt.Errorf("Argument \"PARENT_DIR\" requires at least 1 argument, got 0"),
				WantStderr: "Argument \"PARENT_DIR\" requires at least 1 argument, got 0\n",
				WantData: &command.Data{Values: map[string]interface{}{
					commander.GetwdKey: filepath.FromSlash("/abc/def/ghi"),
				}},
			},
		},
		{
			name:           "parent fails if empty arg",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    "/abc/def/ghi",
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"parent", ""},
				WantData: &command.Data{Values: map[string]interface{}{
					parentDirArg.Name(): "",
					commander.GetwdKey:  filepath.FromSlash("/abc/def/ghi"),
				}},
				WantErr:    fmt.Errorf("PARENT_DIR must be a parent directory"),
				WantStderr: "PARENT_DIR must be a parent directory\n",
			},
		},
		{
			name:           "parent fails if doesn't match",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    "/abc/def/ghi",
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"parent", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					parentDirArg.Name(): "jkl",
					commander.GetwdKey:  filepath.FromSlash("/abc/def/ghi"),
				}},
				WantErr:    fmt.Errorf("PARENT_DIR must be a parent directory"),
				WantStderr: "PARENT_DIR must be a parent directory\n",
			},
		},
		{
			name:           "parent fails if last directory",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    "/abc/def/ghi",
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"parent", "ghi"},
				WantData: &command.Data{Values: map[string]interface{}{
					parentDirArg.Name(): "ghi",
					commander.GetwdKey:  filepath.FromSlash("/abc/def/ghi"),
				}},
				WantErr:    fmt.Errorf("PARENT_DIR must be a parent directory"),
				WantStderr: "PARENT_DIR must be a parent directory\n",
			},
		},
		{
			name:           "parent fails if only one directory",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    "ghi",
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"parent", "ghi"},
				WantData: &command.Data{Values: map[string]interface{}{
					parentDirArg.Name(): "ghi",
					commander.GetwdKey:  filepath.FromSlash("ghi"),
				}},
				WantErr:    fmt.Errorf("PARENT_DIR must be a parent directory"),
				WantStderr: "PARENT_DIR must be a parent directory\n",
			},
		},
		{
			name:           "parent succeeds",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    commandtest.FilepathAbs(t, "abc", "def", "ghi"),
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"parent", "def"},
				WantData: &command.Data{Values: map[string]interface{}{
					parentDirArg.Name(): "def",
					commander.GetwdKey:  commandtest.FilepathAbs(t, "abc", "def", "ghi"),
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf(`cd %q`, commandtest.FilepathAbs(t, "abc", "def"))},
				},
			},
		},
		{
			name:           "parent uses highest level directory if duplciates",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    commandtest.FilepathAbs(t, "abc", "def", "ghi", "def", "jkl"),
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"parent", "def"},
				WantData: &command.Data{Values: map[string]interface{}{
					parentDirArg.Name(): "def",
					commander.GetwdKey:  commandtest.FilepathAbs(t, "abc", "def", "ghi", "def", "jkl"),
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf(`cd %q`, commandtest.FilepathAbs(t, "abc", "def", "ghi", "def"))},
				},
			},
		},
		/* TODO: {
			name:           "parent complexecutes",
			d:              &Dot{},
			wantHistory:    &History{},
			cwdOverride:    "/abc/def/ghi/jkl",
			noShellDataKey: true,
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"parent", "a"},
				WantData: &command.Data{Values: map[string]interface{}{
					parentDirArg.Name(): "a",
					commander.GetwdKey:  filepath.FromSlash("/abc/def/ghi/jkl"),
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{filepath.FromSlash(`cd /abc`)},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.cwdOverride = filepath.FromSlash(test.cwdOverride)
			c := test.shellCache
			if c == nil {
				c = cachetest.NewTestCache(t)
			}
			if test.etc.WantData == nil {
				test.etc.WantData = &command.Data{Values: map[string]interface{}{}}
			}
			if !test.noShellDataKey {
				test.etc.WantData.Values[cache.ShellDataKey] = c
			}
			if test.cwdOverride != "" {
				commandtest.StubGetwd(t, test.cwdOverride, nil)
			} else {
				commandtest.StubGetwd(t, cwd, nil)
			}

			commandtest.StubValue(t, &osStat, func(path string) (os.FileInfo, error) { return test.osStatFI, test.osStatErr })
			cache.StubShellCache(t, c)

			test.etc.Node = test.d.Node()
			test.etc.OS = &commandtest.FakeOS{}
			test.etc.DataCmpOpts = []cmp.Option{
				cmp.AllowUnexported(cache.Cache{}),
			}
			commandertest.ExecuteTest(t, test.etc)
			commandertest.ChangeTest(t, test.want, test.d, cmpopts.IgnoreUnexported(Dot{}), cmpopts.EquateEmpty())

			if !test.ignoreHistoryCheck {
				newH := &History{}
				if _, err := c.GetStruct(shellCacheKey, newH); err != nil {
					t.Fatalf("Failed to read history from cache: %v", err)
				}
				if diff := cmp.Diff(test.wantHistory, newH); diff != "" {
					t.Errorf("Execute(%v) produced incorrect history (-want, +got):\n%s", test.etc.Args, diff)
				}
			}
		})
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name        string
		ctc         *commandtest.CompleteTestCase
		cwdOverride string
	}{
		{
			name: "dot completes all directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Want: &command.Autocompletion{
					Suggestions: []string{
						".git/",
						"cmd/",
						"testing/",
						" ",
					},
				},
			},
		},
		{
			name: "dot completes all directories with command",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd ",
				Want: &command.Autocompletion{
					Suggestions: []string{
						".git/",
						"cmd/",
						"testing/",
						" ",
					},
				},
			},
		},
		{
			name: "dot completes simple directory",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd c",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"cmd/",
					},
					SpacelessCompletion: true,
				},
			},
		},
		{
			name: "dot handles no match",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd uhh",
			},
		},
		{
			name: "dot completes directories that match",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd te",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"testing/",
					},
					SpacelessCompletion: true,
				},
			},
		},
		{
			name: "dot completes nested directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing/o",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"testing/other/",
					},
					SpacelessCompletion: true,
				},
			},
		},
		{
			name: "dot completes sub directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing ",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"dir1/",
						"dir2/",
						"other/",
						" ",
					},
				},
			},
		},
		{
			name: "dot completes sub nested directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing dir1/",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"another/",
						"folderA/",
						"folderB/",
						" ",
					},
				},
			},
		},
		{
			name: "dot completes partial sub nested directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing dir1/fold",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"dir1/folder",
					},
					SpacelessCompletion: true,
				},
			},
		},
		{
			name: "dot completes partial sub directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing d",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"dir",
					},
					SpacelessCompletion: true,
				},
			},
		},
		{
			name: "dot completes partial sub directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing d",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"dir",
					},
					SpacelessCompletion: true,
				},
			},
		},
		{
			name: "dot completion handles no match for sub directories",
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing um",
			},
		},
		{
			name:        "sub directory completion ignores current dir",
			cwdOverride: commandtest.FilepathAbs(t, "testing", "dir1"),
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing ",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"dir2/",
						"other/",
						" ",
					},
				},
			},
		},
		{
			name:        "sub directory completion ignores current dir if nested",
			cwdOverride: commandtest.FilepathAbs(t, "testing", "dir2", "something", "else"),
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing ",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"dir1/",
						"other/",
						" ",
					},
				},
			},
		},
		{
			name:        "parent autocompletes",
			cwdOverride: filepath.FromSlash("/abc/def/ghi/jkl"),
			ctc: &commandtest.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd parent ",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"abc",
						"def",
						"ghi",
					},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.cwdOverride != "" {
				commandtest.StubGetwd(t, test.cwdOverride, nil)
			}

			if test.ctc.Want != nil {
				for i, v := range test.ctc.Want.Suggestions {
					test.ctc.Want.Suggestions[i] = filepath.FromSlash(v)
				}
			}
			test.ctc.SkipDataCheck = true
			test.ctc.OS = &commandtest.FakeOS{}
			commandertest.AutocompleteTest(t, test.ctc)
		})
	}
}

func TestMetadata(t *testing.T) {
	commandtest.StubValue(t, &dotName, ".")
	wantName := "."
	if got := DotCLI().Name(); got != wantName {
		t.Errorf("Name() returned %q; want %q", got, wantName)
	}
}

func TestUsage(t *testing.T) {
	commandertest.ExecuteTest(t, &commandtest.ExecuteTestCase{
		Node: DotCLI().Node(),
		Args: []string{"--help"},
		WantStdout: strings.Join([]string{
			"Changes directories",
			"┳ { shortcuts } [ PATH ] [ SUB_PATH ... ] --up|-u UP",
			"┃",
			"┃   Go to the previous directory",
			"┣━━ -",
			"┃",
			"┣━━ hist",
			"┃",
			"┗━━ parent PARENT_DIR",
			"",
			"Arguments:",
			"  PARENT_DIR: Name of the parent directory to go up to",
			"  PATH: destination directory",
			"  SUB_PATH: subdirectories to continue to",
			"",
			"Flags:",
			"  [u] up: Number of directories to go up when cd-ing",
			"    Default: 0",
			"    NonNegative()",
			"",
			"Symbols:",
			"  { shortcuts }: Start of new shortcut-able section. This is usable by providing the `shortcuts` keyword in this position. Run `cmd ... shortcuts --help` for more details",
			"",
		}, "\n"),
	})
}
