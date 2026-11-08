package gateway

import (
	"context"
	"reflect"
	"testing"

	v1 "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1"
	"github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/fake"
	"github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/types"
)

// withDrafts wraps an SDK client (the fake's GetDraft is unimplemented) so
// that GetDraft returns the chunks of drafts["workspace/name"] that match
// the status filter, or NotFound. GetDraft of the sandbox named fail fails.
type withDrafts struct {
	v1.ClientInterface
	drafts map[string][]v1.PolicyChunk
	fail   string
}

func (c withDrafts) Policy() v1.PolicyInterface {
	return draftPolicy{c.ClientInterface.Policy(), c.drafts, c.fail}
}

type draftPolicy struct {
	v1.PolicyInterface
	drafts map[string][]v1.PolicyChunk
	fail   string
}

func (p draftPolicy) GetDraft(_ context.Context, workspace, name string, opts ...v1.GetDraftOption) (*v1.DraftPolicy, error) {
	if name == p.fail {
		return nil, &types.StatusError{Code: types.ErrorUnavailable, Message: "down"}
	}
	chunks, ok := p.drafts[workspace+"/"+name]
	if !ok {
		return nil, &types.StatusError{Code: types.ErrorNotFound, Message: "sandbox not found"}
	}
	cfg := types.ApplyGetDraftOptions(opts)
	draft := &v1.DraftPolicy{}
	for _, c := range chunks {
		if f := cfg.StatusFilter(); f == "" || c.Status == f {
			draft.Chunks = append(draft.Chunks, c)
		}
	}
	return draft, nil
}

// chunk is a draft chunk as the test gateway proposes it after apt-get
// was denied.
func chunk(id, status, host string, port uint32, binary string) v1.PolicyChunk {
	return v1.PolicyChunk{
		ID:     id,
		Status: status,
		Binary: binary,
		ProposedRule: &v1.NetworkPolicyRule{
			Endpoints: []v1.PolicyNetworkEndpoint{{Host: host, Port: port, Ports: []uint32{port}}},
		},
	}
}

func TestProposals(t *testing.T) {
	multi := chunk("b", "pending", "", 0, "/usr/bin/node")
	multi.ProposedRule.Endpoints = []v1.PolicyNetworkEndpoint{
		{Host: "api.example.com", Ports: []uint32{443, 8443}},
		{Host: "::1", Ports: []uint32{80}},
	}
	got := proposals([]v1.PolicyChunk{
		chunk("c", "pending", "ports.ubuntu.com", 80, "/usr/lib/apt/methods/http"),
		multi,
		{ID: "a", Status: "pending", Binary: "/bin/true"}, // no proposed rule
	})
	want := []PolicyProposal{ // sorted by ID
		{ID: "a", Binary: "/bin/true"},
		{ID: "b", Endpoints: []string{"api.example.com:443", "api.example.com:8443", "[::1]:80"}, Binary: "/usr/bin/node"},
		{ID: "c", Endpoints: []string{"ports.ubuntu.com:80"}, Binary: "/usr/lib/apt/methods/http"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("proposals =\n%+v\nwant\n%+v", got, want)
	}
}

func TestAttachProposals(t *testing.T) {
	ctx := context.Background()
	apt := chunk("c1", "pending", "ports.ubuntu.com", 80, "/usr/lib/apt/methods/http")
	c := withDrafts{ClientInterface: fake.NewClient(), drafts: map[string][]v1.PolicyChunk{
		"default/a": {apt, chunk("c2", "rejected", "evil.example", 443, "/usr/bin/curl")},
	}}

	t.Run("pending only, sandbox gone", func(t *testing.T) {
		sbs := []Sandbox{{Name: "a", Workspace: "default"}, {Name: "gone", Workspace: "default"}}
		if err := attachProposals(ctx, c, sbs); err != nil {
			t.Fatal(err)
		}
		want := []Sandbox{
			{Name: "a", Workspace: "default", Proposals: []PolicyProposal{{ID: "c1", Endpoints: []string{"ports.ubuntu.com:80"}, Binary: apt.Binary}}},
			{Name: "gone", Workspace: "default"},
		}
		if !reflect.DeepEqual(sbs, want) {
			t.Errorf("sandboxes =\n%+v\nwant\n%+v", sbs, want)
		}
	})
	t.Run("error leaves all sandboxes without proposals", func(t *testing.T) {
		c := c
		c.fail = "b"
		sbs := []Sandbox{{Name: "a", Workspace: "default"}, {Name: "b", Workspace: "default"}}
		if err := attachProposals(ctx, c, sbs); err == nil || sbs[0].Proposals != nil {
			t.Errorf("attachProposals = %v, sandboxes %+v", err, sbs)
		}
	})
}
