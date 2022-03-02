package cd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leep-frog/command"
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
			name: "handles empty string",
		},
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := &Dot{}
			if err := d.Load(test.json); err != nil {
				t.Fatalf("Load(%v) should return nil; got %v", test.json, err)
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

func TestExecution(t *testing.T) {
	for _, test := range []struct {
		name      string
		d         *Dot
		etc       *command.ExecuteTestCase
		osStatFI  os.FileInfo
		osStatErr error
	}{
		{
			name:     "handles nil arguments",
			osStatFI: dirType,
			d:        DotCLI(1),
			etc: &command.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd .."},
				},
			},
		},
		{
			name:     "handles basic dot",
			osStatFI: dirType,
			d:        DotCLI(0),
			etc: &command.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"cd "},
				},
			},
		},
		{
			name:     "handles empty arguments",
			osStatFI: dirType,
			d:        DotCLI(2),
			etc: &command.ExecuteTestCase{
				Args: []string{},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %s", fp(filepath.Join("..", ".."))),
					},
				},
			},
		},
		{
			name:     "handles absolute path",
			osStatFI: dirType,
			d:        DotCLI(0),
			etc: &command.ExecuteTestCase{
				Args: []string{filepathAbs(t, "../../..")},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %s", fp(filepathAbs(t, filepath.Join("..", "..", "..")))),
					},
				},

				WantData: &command.Data{Values: map[string]interface{}{
					pathArg: filepathAbs(t, filepath.Join("..", "..", "..")),
				}},
			},
		},
		{
			name:     "cds into directory of a file",
			osStatFI: fileType,
			d:        DotCLI(3),
			etc: &command.ExecuteTestCase{
				Args: []string{"something/somewhere.txt"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %s", fp(filepathAbs(t, filepath.Join("..", "..", "..", "something")))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg: filepathAbs(t, filepath.Join("..", "..", "..", "something", "somewhere.txt")),
				}},
			},
		},
		{
			name:     "cds into directory with spaces",
			osStatFI: dirType,
			d:        DotCLI(0),
			etc: &command.ExecuteTestCase{
				Args: []string{"some where/"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %s", fp(filepathAbs(t, filepath.Join("some where")))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg: filepathAbs(t, filepath.Join("some where")),
				}},
			},
		},
		{
			name:     "0-dot cds down multiple paths",
			osStatFI: dirType,
			d:        DotCLI(0),
			etc: &command.ExecuteTestCase{
				Args: []string{"some", "thing", "some", "where"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %s", fp(filepathAbs(t, filepath.Join("some", "thing", "some", "where")))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:    filepathAbs(t, filepath.Join("some")),
					subPathArg: []string{"thing", "some", "where"},
				}},
			},
		},
		{
			name:     "1-dot cds down multiple paths",
			osStatFI: dirType,
			d:        DotCLI(1),
			etc: &command.ExecuteTestCase{
				Args: []string{"some", "thing", "some", "where"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("cd %s", fp(filepathAbs(t, filepath.Join("..", "some", "thing", "some", "where")))),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArg:    filepathAbs(t, filepath.Join("..", "some")),
					subPathArg: []string{"thing", "some", "where"},
				}},
			},
		},
		{
			name: "0-dot goes to the previous directory",
			d:    DotCLI(0),
			etc: &command.ExecuteTestCase{
				Args: []string{"-"},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"cd -",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			oldStat := osStat
			osStat = func(path string) (os.FileInfo, error) { return test.osStatFI, test.osStatErr }
			defer func() { osStat = oldStat }()

			test.etc.Node = test.d.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, nil, test.d)
		})
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name string
		ctc  *command.CompleteTestCase
	}{
		{
			name: "dot completes all directories",
			ctc: &command.CompleteTestCase{
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
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
			name: "dot handles no match",
			ctc: &command.CompleteTestCase{
				Node: DotCLI(0).Node(),
				Args: "cmd uhh",
			},
		},
		{
			name: "dot completes directories that match",
			ctc: &command.CompleteTestCase{
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
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
				Node: DotCLI(0).Node(),
				Args: "cmd testing um",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.ctc.SkipDataCheck = true
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestMetadata(t *testing.T) {
	d := DotCLI(4)

	wantName := "....."
	if got := d.Name(); got != wantName {
		t.Errorf("Name() returned %q; want %q", got, wantName)
	}
}

func TestUsage(t *testing.T) {
	// Test with single dot
	command.UsageTest(t, &command.UsageTestCase{
		Node: DotCLI(0).Node(),
		WantString: []string{
			"Changes directories",
			"< * [ PATH ] [ SUB_PATH ... ]",
			"",
			"  Go to the previous directory",
			"  -",
			"",
			"Arguments:",
			"  PATH: destination directory",
			"  SUB_PATH: subdirectories to continue to",
			"",
			"Symbols:",
			command.AliasDesc,
			command.BranchDesc,
		},
	})

	// Test with multiple dots
	command.UsageTest(t, &command.UsageTestCase{
		Node: DotCLI(1).Node(),
		WantString: []string{
			"Changes directories",
			"[ PATH ] [ SUB_PATH ... ]",
			"",
			"Arguments:",
			"  PATH: destination directory",
			"  SUB_PATH: subdirectories to continue to",
		},
	})
}
