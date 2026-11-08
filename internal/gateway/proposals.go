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

// PolicyProposal is a pending chunk of a sandbox's policy draft: a rule the
// gateway suggests after the sandbox's policy denied network requests.
type PolicyProposal struct {
	ID        string   // chunk ID
	Endpoints []string // host:port that was denied
	Binary    string   // the binary that made the request
}

// attachProposals reads the pending policy proposals of each sandbox. If
// one read fails, no sandbox gets proposals.
func attachProposals(ctx context.Context, sdk v1.ClientInterface, sbs []Sandbox) error {
	all := make([][]PolicyProposal, len(sbs))
	for i, sb := range sbs {
		ps, err := fetchProposals(ctx, sdk, sb.Workspace, sb.Name)
		if err != nil {
			return err
		}
		all[i] = ps
	}
	for i := range sbs {
		sbs[i].Proposals = all[i]
	}
	return nil
}

// fetchProposals reads the pending draft chunks of a sandbox. A sandbox
// deleted since it was listed has none.
func fetchProposals(ctx context.Context, sdk v1.ClientInterface, workspace, name string) ([]PolicyProposal, error) {
	draft, err := sdk.Policy().GetDraft(ctx, workspace, name, v1.WithStatusFilter("pending"))
	if v1.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("policy draft of %s/%s: %w", workspace, name, err)
	}
	return proposals(draft.Chunks), nil
}

// proposals turns chunks into proposals, sorted by ID so that a poll round
// with the same chunks compares equal.
func proposals(chunks []v1.PolicyChunk) []PolicyProposal {
	var props []PolicyProposal
	for _, c := range chunks {
		prop := PolicyProposal{ID: c.ID, Binary: c.Binary}
		if c.ProposedRule != nil {
			for _, ep := range c.ProposedRule.Endpoints {
				for _, p := range ep.Ports {
					prop.Endpoints = append(prop.Endpoints, net.JoinHostPort(ep.Host, strconv.FormatUint(uint64(p), 10)))
				}
			}
		}
		props = append(props, prop)
	}
	slices.SortFunc(props, func(a, b PolicyProposal) int { return cmp.Compare(a.ID, b.ID) })
	return props
}
