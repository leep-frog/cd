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
		name       string
		d          *Dot
		args       []string
		wantEData  *command.ExecuteData
		wantErr    error
		wantData   *command.Data
		wantStdout []string
		wantStderr []string
		osStatFI   os.FileInfo
		osStatErr  error
	}{
		{
			name:     "handles nil arguments",
			d:        DotCLI(1),
			osStatFI: dirType,
			wantEData: &command.ExecuteData{
				Executable: [][]string{{"cd", ".."}},
			},
		},
		{
			name:     "handles empty arguments",
			d:        DotCLI(2),
			osStatFI: dirType,
			args:     []string{},
			wantEData: &command.ExecuteData{
				Executable: [][]string{{
					"cd",
					filepath.Join("../../"),
				}},
			},
		},
		{
			name:     "cds into directory of a file",
			d:        DotCLI(3),
			osStatFI: fileType,
			args:     []string{"something/somewhere.txt"},
			wantEData: &command.ExecuteData{
				Executable: [][]string{{
					"cd",
					filepath.Join("..", "..", "..", "something"),
				}},
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					"path": command.StringValue("something/somewhere.txt"),
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			oldStat := osStat
			osStat = func(path string) (os.FileInfo, error) { return test.osStatFI, test.osStatErr }
			defer func() { osStat = oldStat }()

			command.ExecuteTest(t, test.d.Node(), test.args, test.wantErr, test.wantEData, test.wantData, test.wantStdout, test.wantStderr)
			if test.d.Changed() {
				t.Fatalf("Execute(%v) marked Changed as true; want false", test.args)
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
