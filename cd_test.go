package cd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leep-frog/command"
)

func TestLoad(t *testing.T) {
	for _, test := range []struct {
		name string
		json string
	}{
		{
			name: "handles empty string",
		},
		{
			name: "handles invalid json",
			json: "}}",
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
					Executable: [][]string{{"cd", ".."}},
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
					Executable: [][]string{{
						"cd",
						filepath.Join("../../"),
					}},
				},
			},
		},
		{
			name:     "cds into directory of a file",
			osStatFI: fileType,
			d:        DotCLI(3),
			etc: &command.ExecuteTestCase{
				Args: []string{"something/somewhere.txt"},
				WantExecuteData: &command.ExecuteData{
					Executable: [][]string{{
						"cd",
						filepath.Join("..", "..", "..", "something"),
					}},
				},
				WantData: &command.Data{
					Values: map[string]*command.Value{
						"path": command.StringValue("something/somewhere.txt"),
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
			command.ExecuteTest(t, test.etc, nil)
			if test.d.Changed() {
				t.Fatalf("Execute(%v) marked Changed as true; want false", test.etc.Args)
			}
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
