package gateway

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// Forward is a running local port forward to a sandbox that the openshell
// CLI tracks (`openshell forward start`), as in `openshell forward list -o
// json`.
type Forward struct {
	Workspace string `json:"workspace"`
	Sandbox   string `json:"sandbox"`
	Bind      string `json:"bind_address"` // local bind address, e.g. 127.0.0.1
	Port      int    `json:"port"`
}

// cliCandidates are tried when openshell is not on PATH: an app started
// from the Finder does not get the shell's PATH.
var cliCandidates = []string{"/opt/homebrew/bin/openshell", "/usr/local/bin/openshell"}

// cliPath finds the openshell binary.
func cliPath() (string, error) {
	if p, err := exec.LookPath("openshell"); err == nil {
		return p, nil
	}
	for _, p := range cliCandidates {
		if _, err := exec.LookPath(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("openshell CLI not found on PATH or in " + strings.Join(cliCandidates, ", "))
}

// listForwards returns the running forwards from `openshell forward list -o
// json`. Forwards are local CLI state; the gateway does not know them. The
// CLI also lists forwards whose process has died; those are left out.
func listForwards(ctx context.Context) ([]Forward, error) {
	bin, err := cliPath()
	if err != nil {
		return nil, err
	}
	out, err := exec.CommandContext(ctx, bin, "forward", "list", "-o", "json").Output()
	if err != nil {
		if ee, ok := errors.AsType[*exec.ExitError](err); ok && len(ee.Stderr) > 0 {
			err = fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("openshell forward list: %w", err)
	}
	var all []struct {
		Forward
		Alive bool `json:"alive"`
	}
	if err := json.Unmarshal(out, &all); err != nil {
		return nil, fmt.Errorf("openshell forward list: %w", err)
	}
	var fwds []Forward
	for _, f := range all {
		if f.Alive {
			fwds = append(fwds, f.Forward)
		}
	}
	return fwds, nil
}

// attachForwards adds each forward to its sandbox, sorted by port. Forwards
// to sandboxes the gateway does not list are dropped: the CLI keeps forwards
// of all gateways together.
func attachForwards(sbs []Sandbox, fwds []Forward) {
	slices.SortFunc(fwds, func(a, b Forward) int {
		return cmp.Or(cmp.Compare(a.Port, b.Port), cmp.Compare(a.Bind, b.Bind))
	})
	for i := range sbs {
		sb := &sbs[i]
		for _, f := range fwds {
			if f.Workspace == sb.Workspace && f.Sandbox == sb.Name {
				sb.Forwards = append(sb.Forwards, f)
			}
		}
	}
}
