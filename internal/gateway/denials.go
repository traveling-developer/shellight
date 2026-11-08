package gateway

import (
	"cmp"
	"context"
	"fmt"
	"net"
	"slices"
	"strconv"

	v1 "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1"
)

// Denial is a pending rule the gateway proposes after it denied network
// requests of a sandbox (a chunk of the sandbox's draft policy).
type Denial struct {
	ID        string   // chunk ID
	Endpoints []string // host:port that was denied
	Binary    string   // the binary that made the request
}

// attachDenials reads the pending denials of each sandbox. If one read
// fails, no sandbox gets denials.
func attachDenials(ctx context.Context, sdk v1.ClientInterface, sbs []Sandbox) error {
	all := make([][]Denial, len(sbs))
	for i, sb := range sbs {
		ds, err := fetchDenials(ctx, sdk, sb.Workspace, sb.Name)
		if err != nil {
			return err
		}
		all[i] = ds
	}
	for i := range sbs {
		sbs[i].Denials = all[i]
	}
	return nil
}

// fetchDenials reads the pending draft chunks of a sandbox. A sandbox
// deleted since it was listed has none.
func fetchDenials(ctx context.Context, sdk v1.ClientInterface, workspace, name string) ([]Denial, error) {
	draft, err := sdk.Policy().GetDraft(ctx, workspace, name, v1.WithStatusFilter("pending"))
	if v1.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("policy draft of %s/%s: %w", workspace, name, err)
	}
	return denials(draft.Chunks), nil
}

// denials turns chunks into denials, sorted by ID so that a poll round
// with the same chunks compares equal.
func denials(chunks []v1.PolicyChunk) []Denial {
	var ds []Denial
	for _, c := range chunks {
		d := Denial{ID: c.ID, Binary: c.Binary}
		if c.ProposedRule != nil {
			for _, ep := range c.ProposedRule.Endpoints {
				for _, p := range ep.Ports {
					d.Endpoints = append(d.Endpoints, net.JoinHostPort(ep.Host, strconv.FormatUint(uint64(p), 10)))
				}
			}
		}
		ds = append(ds, d)
	}
	slices.SortFunc(ds, func(a, b Denial) int { return cmp.Compare(a.ID, b.ID) })
	return ds
}
