package gateway

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// fakeCLI puts an openshell script on PATH that prints out on stdout,
// errOut on stderr and exits with code. out and errOut must not contain
// single quotes.
func fakeCLI(t *testing.T, out, errOut string, code int) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"[ \"$*\" = 'forward list -o json' ] || { echo \"unexpected args: $*\" >&2; exit 64; }\n" +
		"printf '%s' '" + out + "'\n" +
		"printf '%s' '" + errOut + "' >&2\n" +
		"exit " + strconv.Itoa(code) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "openshell"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestListForwards(t *testing.T) {
	// As printed by openshell 0.1.7; the dead forward is left out.
	fakeCLI(t, `[{"alive":true,"bind_address":"127.0.0.1","pid":83545,"port":18080,"sandbox":"fwtest","workspace":"default"},`+
		`{"alive":false,"bind_address":"127.0.0.1","pid":83546,"port":18081,"sandbox":"fwtest","workspace":"default"}]`, "", 0)
	got, err := listForwards(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Forward{{Workspace: "default", Sandbox: "fwtest", Bind: "127.0.0.1", Port: 18080}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("listForwards = %+v, want %+v", got, want)
	}
}

func TestListForwardsErrors(t *testing.T) {
	t.Run("CLI fails", func(t *testing.T) {
		fakeCLI(t, "", "boom", 1)
		_, err := listForwards(context.Background())
		if err == nil || !strings.Contains(err.Error(), "boom") {
			t.Errorf("err = %v, want stderr in it", err)
		}
	})
	t.Run("CLI not found", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		old := cliCandidates
		cliCandidates = []string{filepath.Join(t.TempDir(), "openshell")}
		t.Cleanup(func() { cliCandidates = old })
		if _, err := listForwards(context.Background()); err == nil {
			t.Error("no error")
		}
	})
}

func TestAttachForwards(t *testing.T) {
	sbs := []Sandbox{{Name: "a", Workspace: "default"}, {Name: "a", Workspace: "team"}, {Name: "b", Workspace: "default"}}
	a9000 := Forward{Workspace: "default", Sandbox: "a", Bind: "127.0.0.1", Port: 9000}
	a8080 := Forward{Workspace: "default", Sandbox: "a", Bind: "0.0.0.0", Port: 8080}
	team := Forward{Workspace: "team", Sandbox: "a", Bind: "127.0.0.1", Port: 3000}
	gone := Forward{Workspace: "default", Sandbox: "gone", Bind: "127.0.0.1", Port: 5000} // other gateway
	attachForwards(sbs, []Forward{a9000, a8080, team, gone})
	want := []Sandbox{
		{Name: "a", Workspace: "default", Forwards: []Forward{a8080, a9000}}, // sorted by port
		{Name: "a", Workspace: "team", Forwards: []Forward{team}},
		{Name: "b", Workspace: "default"},
	}
	if !reflect.DeepEqual(sbs, want) {
		t.Errorf("sandboxes =\n%+v\nwant\n%+v", sbs, want)
	}
}
