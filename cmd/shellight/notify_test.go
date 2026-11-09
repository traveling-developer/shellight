package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/traveling-developer/shellight/internal/gateway"
)

// proposal is a policy proposal for ports.ubuntu.com:80.
func proposal(id, supersedes string) gateway.PolicyProposal {
	return gateway.PolicyProposal{ID: id, Endpoints: []string{"ports.ubuntu.com:80"}, Supersedes: supersedes}
}

// withProposals is a status of gateway gw whose sandbox fwtest has ps.
func withProposals(gw string, ps ...gateway.PolicyProposal) gateway.Status {
	return gateway.Status{Name: gw, Info: gateway.Info{Sandboxes: []gateway.Sandbox{{Name: "fwtest", Proposals: ps}}}}
}

// ids returns the proposal IDs per sandbox of notices.
func ids(notices []notice) map[string][]string {
	if len(notices) == 0 {
		return nil
	}
	m := make(map[string][]string)
	for _, n := range notices {
		for _, p := range n.Proposals {
			m[n.Sandbox] = append(m[n.Sandbox], p.ID)
		}
	}
	return m
}

func TestProposalTracker(t *testing.T) {
	type round struct {
		s    gateway.Status
		want map[string][]string
	}
	tests := []struct {
		name   string
		rounds []round
	}{
		{"pending at start does not notify", []round{
			{withProposals("gw", proposal("c1", "")), nil},
			{withProposals("gw", proposal("c1", "")), nil},
		}},
		{"new proposals notify once, bundled per sandbox", []round{
			{withProposals("gw", proposal("c1", "")), nil},
			{withProposals("gw", proposal("c1", ""), proposal("c2", ""), proposal("c3", "")),
				map[string][]string{"fwtest": {"c2", "c3"}}},
			{withProposals("gw", proposal("c1", ""), proposal("c2", ""), proposal("c3", "")), nil},
		}},
		{"one notice per sandbox", []round{
			{gateway.Status{Name: "gw"}, nil},
			{gateway.Status{Name: "gw", Info: gateway.Info{Sandboxes: []gateway.Sandbox{
				{Name: "a", Proposals: []gateway.PolicyProposal{proposal("c1", "")}},
				{Name: "b", Proposals: []gateway.PolicyProposal{proposal("c2", "")}},
			}}}, map[string][]string{"a": {"c1"}, "b": {"c2"}}},
		}},
		{"refined chunk of a known one does not notify", []round{
			{withProposals("gw", proposal("c1", "")), nil},
			{withProposals("gw", proposal("c2", "c1")), nil},
		}},
		{"second refinement of a known one does not notify", []round{
			{withProposals("gw", proposal("c1", "")), nil},
			{withProposals("gw", proposal("c2", "c1")), nil},
			{withProposals("gw", proposal("c3", "c1")), nil},
		}},
		{"refined chunk of an unknown one notifies", []round{
			{withProposals("gw"), nil},
			{withProposals("gw", proposal("c2", "c1")), map[string][]string{"fwtest": {"c2"}}},
		}},
		{"error round changes nothing", []round{
			{withProposals("gw"), nil},
			{gateway.Status{Name: "gw", Err: errors.New("down")}, nil},
			{withProposals("gw", proposal("c1", "")), map[string][]string{"fwtest": {"c1"}}},
		}},
		{"proposal error round changes nothing", []round{
			{withProposals("gw"), nil},
			{gateway.Status{Name: "gw", ProposalErr: errors.New("draft")}, nil},
			{withProposals("gw", proposal("c1", "")), map[string][]string{"fwtest": {"c1"}}},
		}},
		{"first round is an error", []round{
			{gateway.Status{Err: errors.New("no config")}, nil},
			{withProposals("gw", proposal("c1", "")), nil},
		}},
		{"gateway switch does not notify", []round{
			{withProposals("gw"), nil},
			{withProposals("other", proposal("c1", "")), nil},
			{withProposals("other", proposal("c1", ""), proposal("c2", "")), map[string][]string{"fwtest": {"c2"}}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tr proposalTracker
			for i, r := range tt.rounds {
				if got := ids(tr.update(r.s)); !reflect.DeepEqual(got, r.want) {
					t.Errorf("round %d: new = %v, want %v", i+1, got, r.want)
				}
			}
		})
	}
}

func TestNoticeBody(t *testing.T) {
	tests := []struct {
		name string
		ps   []gateway.PolicyProposal
		want string
	}{
		{"one", []gateway.PolicyProposal{proposal("c1", "")}, "New policy proposal: ports.ubuntu.com:80"},
		{"one without endpoint", []gateway.PolicyProposal{{ID: "c1"}}, "New policy proposal"},
		{"several", []gateway.PolicyProposal{proposal("c1", ""), proposal("c2", ""), proposal("c3", "")}, "3 new policy proposals"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (notice{Sandbox: "fwtest", Proposals: tt.ps}).body(); got != tt.want {
				t.Errorf("body = %q, want %q", got, tt.want)
			}
		})
	}
}
