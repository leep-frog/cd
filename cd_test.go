package cd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/commands/commands"
)

func TestLoad(t *testing.T) {
	for _, test := range []struct {
		name string
		json string
		want *Dot
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
		args      []string
		wantResp  *commands.ExecutorResponse
		wantErr   string
		wantOK bool
		wantStdout []string
		wantStderr []string
		osStatFI  os.FileInfo
		osStatErr error
	}{
		{
			name:     "handles nil arguments",
			d:        DotCLI(1),
			osStatFI: dirType,
			wantOK: true,
			wantResp: &commands.ExecutorResponse{
				Executable: []string{"cd", ".."},
			},
		},
		{
			name:     "handles empty arguments",
			d:        DotCLI(2),
			osStatFI: dirType,
			args:     []string{},
			wantOK: true,
			wantResp: &commands.ExecutorResponse{
				Executable: []string{
					"cd",
					filepath.Join("../../"),
				},
			},
		},
		{
			name:     "cds into directory of a file",
			d:        DotCLI(3),
			osStatFI: fileType,
			args:     []string{"something/somewhere.txt"},
			wantOK: true,
			wantResp: &commands.ExecutorResponse{
				Executable: []string{
					"cd",
					filepath.Join("..", "..", "..", "something"),
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			oldStat := osStat
			osStat = func(path string) (os.FileInfo, error) { return test.osStatFI, test.osStatErr }
			defer func() { osStat = oldStat }()

			tcos := &commands.TestCommandOS{}
			got, ok := commands.Execute(tcos, test.d.Command(), test.args, nil)
			if ok != test.wantOK {
				t.Fatalf("Execute(%v, %v) returned %v for ok; want %v", test.d.Command(), test.args, ok, test.wantOK)
			}

			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("Execute(%v) produced response diff (-want, +got):\n%s", test.args, diff)
			}

			if diff := cmp.Diff(test.wantStdout, tcos.GetStdout()); diff != "" {
				t.Errorf("command.Execute(%v) produced stdout diff (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, tcos.GetStderr()); diff != "" {
				t.Errorf("command.Execute(%v) produced stderr diff (-want, +got):\n%s", test.args, diff)
			}

			if test.d.Changed() {
				t.Fatalf("Execute(%v) marked Changed as true; want false", test.args)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	d := DotCLI(4)

	wantName := "4-dir-dot"
	if got := d.Name(); got != wantName {
		t.Fatalf("Name() returned %q; want %q", got, wantName)
	}

	wantAlias := "....."
	if got := d.Alias(); got != wantAlias {
		t.Fatalf("Alias() returned %q; want %q", got, wantAlias)
	}
}
