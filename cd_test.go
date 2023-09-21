package cd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
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

	command.StubValue(t, &dotName, ".")

	for _, test := range []struct {
		name               string
		d                  *Dot
		want               *Dot
		etc                *command.ExecuteTestCase
		osStatFI           os.FileInfo
		osStatErr          error
		shellCache         *cache.Cache
		ignoreHistoryCheck bool
		wantHistory        *History
		cwdOverride        string
	}{
		{
			name:        "handles nil arguments",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():    0,
						command.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:     "error if GetStruct error",
			osStatFI: dirType,
			d:        DotCLI(),
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: "} invalid json {",
			}),
			ignoreHistoryCheck: true,
			wantHistory:        &History{},
			etc: &command.ExecuteTestCase{
				WantErr:    fmt.Errorf("failed to get struct data: failed to unmarshal cache data: invalid character '}' looking for beginning of value"),
				WantStderr: "failed to get struct data: failed to unmarshal cache data: invalid character '}' looking for beginning of value\n",
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():    0,
						command.GetwdKey: cwd,
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
			etc: &command.ExecuteTestCase{
				Args: []string{"c"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "cmd"))},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():    0,
						"PATH":           filepathAbs(t, "cmd"),
						command.GetwdKey: filepathAbs(t, "."),
					},
				},
			},
		},
		{
			name:        "handles basic dot",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():    0,
						command.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles empty arguments",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():    0,
						command.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles directory with spaces arguments",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{"some thing"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, "some thing")),
					},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						pathArg:          filepathAbs(t, "some thing"),
						upFlag.Name():    0,
						command.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles -u flag",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{"-u", "2"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepath.Join("..", "..")),
					},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						upFlag.Name():    2,
						command.GetwdKey: cwd,
					},
				},
			},
		},
		{
			name:        "handles absolute path",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{filepathAbs(t, "../../..")},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("..", "..", ".."))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:          filepathAbs(t, filepath.Join("..", "..", "..")),
					upFlag.Name():    0,
					command.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "cds into directory of a file",
			osStatFI:    fileType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{"something/somewhere.txt", "--up", "3"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("..", "..", "..", "something"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:          filepathAbs(t, filepath.Join("..", "..", "..", "something", "somewhere.txt")),
					upFlag.Name():    3,
					command.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "cds into directory with spaces",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{"some where/"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("some where"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:          filepathAbs(t, filepath.Join("some where")),
					upFlag.Name():    0,
					command.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "0-dot cds down multiple paths",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{"some", "thing", "some", "where"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("some", "thing", "some", "where"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:          filepathAbs(t, filepath.Join("some")),
					subPathArg:       []string{"thing", "some", "where"},
					upFlag.Name():    0,
					command.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "1-dot cds down multiple paths",
			osStatFI:    dirType,
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{"some", "thing", "-u", "1", "some", "where"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %q", filepathAbs(t, filepath.Join("..", "some", "thing", "some", "where"))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:          filepathAbs(t, filepath.Join("..", "some")),
					subPathArg:       []string{"thing", "some", "where"},
					upFlag.Name():    1,
					command.GetwdKey: cwd,
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
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir",
					},
				},
			}),
			etc: &command.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
				}},
			},
		},
		{
			name:        "minus goes home if no history",
			d:           DotCLI(),
			wantHistory: wdHist,
			etc: &command.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
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
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
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
			etc: &command.ExecuteTestCase{
				Args: []string{"somewhere"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "somewhere"))},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
					"PATH":           filepathAbs(t, "somewhere"),
					upFlag.Name():    0,
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
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
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
			etc: &command.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir/5"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
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
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
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
			etc: &command.ExecuteTestCase{
				Args: []string{"somewhere"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "somewhere"))},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
					"PATH":           filepathAbs(t, "somewhere"),
					upFlag.Name():    0,
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
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
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
			etc: &command.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir/2"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
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
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						cwd,
					},
				},
			}),
			etc: &command.ExecuteTestCase{
				Args: []string{"somewhere"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{fmt.Sprintf("cd %q", filepathAbs(t, "somewhere"))},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
					"PATH":           filepathAbs(t, "somewhere"),
					upFlag.Name():    0,
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
			shellCache: cache.NewTestCacheWithData(t, map[string]interface{}{
				shellCacheKey: &History{
					PrevDirs: []string{
						"old/dir/1",
						cwd,
					},
				},
			}),
			etc: &command.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`cd "old/dir/1"`,
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.GetwdKey: cwd,
				}},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			c := test.shellCache
			if c == nil {
				c = cache.NewTestCache(t)
			}
			if test.etc.WantData == nil {
				test.etc.WantData = &command.Data{Values: map[string]interface{}{}}
			}
			test.etc.WantData.Values[cache.ShellDataKey] = c
			if test.cwdOverride != "" {
				command.StubGetwd(t, test.cwdOverride, nil)
			} else {
				command.StubGetwd(t, cwd, nil)
			}

			command.StubValue(t, &osStat, func(path string) (os.FileInfo, error) { return test.osStatFI, test.osStatErr })
			cache.StubShellCache(t, c)

			test.etc.Node = test.d.Node()
			test.etc.OS = &command.FakeOS{}
			test.etc.DataCmpOpts = []cmp.Option{
				cmp.AllowUnexported(cache.Cache{}),
			}
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, test.want, test.d, cmpopts.IgnoreUnexported(Dot{}), cmpopts.EquateEmpty())

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
		ctc         *command.CompleteTestCase
		cwdOverride string
	}{
		{
			name: "dot completes all directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Want: []string{
					".git/",
					"cmd/",
					"testing/",
					" ",
				},
			},
		},
		{
			name: "dot completes all directories with command",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd ",
				Want: []string{
					".git/",
					"cmd/",
					"testing/",
					" ",
				},
			},
		},
		{
			name: "dot completes simple directory",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd c",
				Want: []string{
					"cmd/",
					"cmd/_",
				},
			},
		},
		{
			name: "dot handles no match",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd uhh",
			},
		},
		{
			name: "dot completes directories that match",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd te",
				Want: []string{
					"testing/",
					"testing/_",
				},
			},
		},
		{
			name: "dot completes nested directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing/o",
				Want: []string{
					"testing/other/",
					"testing/other/_",
				},
			},
		},
		{
			name: "dot completes sub directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing ",
				Want: []string{
					"dir1/",
					"dir2/",
					"other/",
					" ",
				},
			},
		},
		{
			name: "dot completes sub nested directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing dir1/",
				Want: []string{
					"another/",
					"folderA/",
					"folderB/",
					" ",
				},
			},
		},
		{
			name: "dot completes partial sub nested directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing dir1/fold",
				Want: []string{
					"dir1/folder",
					"dir1/folder_",
				},
			},
		},
		{
			name: "dot completes partial sub directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing d",
				Want: []string{
					"dir",
					"dir_",
				},
			},
		},
		{
			name: "dot completes partial sub directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing d",
				Want: []string{
					"dir",
					"dir_",
				},
			},
		},
		{
			name: "dot completion handles no match for sub directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing um",
			},
		},
		{
			name:        "sub directory completion ignores current dir",
			cwdOverride: command.FilepathAbs(t, "testing", "dir1"),
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing ",
				Want: []string{
					"dir2/",
					"other/",
					" ",
				},
			},
		},
		{
			name:        "sub directory completion ignores current dir if nested",
			cwdOverride: command.FilepathAbs(t, "testing", "dir2", "something", "else"),
			ctc: &command.CompleteTestCase{
				Node: DotCLI().Node(),
				Args: "cmd testing ",
				Want: []string{
					"dir1/",
					"other/",
					" ",
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.cwdOverride != "" {
				command.StubGetwd(t, test.cwdOverride, nil)
			}
			test.ctc.SkipDataCheck = true
			test.ctc.OS = &command.FakeOS{}
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestMetadata(t *testing.T) {
	command.StubValue(t, &dotName, ".")
	wantName := "."
	if got := DotCLI().Name(); got != wantName {
		t.Errorf("Name() returned %q; want %q", got, wantName)
	}
}

func TestUsage(t *testing.T) {
	command.UsageTest(t, &command.UsageTestCase{
		Node: DotCLI().Node(),
		WantString: []string{
			"Changes directories",
			"< * [ PATH ] [ SUB_PATH ... ] --up|-u",
			"",
			"  Go to the previous directory",
			"  -",
			"",
			"  hist",
			"",
			"Arguments:",
			"  PATH: destination directory",
			"  SUB_PATH: subdirectories to continue to",
			"",
			"Flags:",
			"  [u] up: Number of directories to go up when cd-ing",
			"",
			"Symbols:",
			command.ShortcutDesc,
			command.BranchDesc,
		},
	})
}
