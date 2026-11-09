package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/traveling-developer/shellight/internal/gateway"
)

// proposalTracker finds the policy proposals that are new since the last
// poll round.
type proposalTracker struct {
	gateway string          // the gateway the seen IDs belong to
	seen    map[string]bool // chunk IDs; nil before the first round
}

// notice is one notification: the new policy proposals of one sandbox.
type notice struct {
	Sandbox   string
	Proposals []gateway.PolicyProposal
}

// update returns the new policy proposals of s, one notice per sandbox. The
// first round, and the first round after a gateway switch, only remember
// what is pending. A refined chunk that replaces a known one is not new;
// it names the initial chunk, so that ID stays known while the refined one
// is pending. A round with an error changes nothing.
func (t *proposalTracker) update(s gateway.Status) []notice {
	if s.Err != nil || s.ProposalErr != nil {
		return nil
	}
	first := t.seen == nil || t.gateway != s.Name
	seen := make(map[string]bool)
	var notices []notice
	for _, sb := range s.Info.Sandboxes {
		var fresh []gateway.PolicyProposal
		for _, p := range sb.Proposals {
			seen[p.ID] = true
			if p.Supersedes != "" {
				seen[p.Supersedes] = true
			}
			if !first && !t.seen[p.ID] && (p.Supersedes == "" || !t.seen[p.Supersedes]) {
				fresh = append(fresh, p)
			}
		}
		if len(fresh) > 0 {
			notices = append(notices, notice{Sandbox: sb.Name, Proposals: fresh})
		}
	}
	t.gateway, t.seen = s.Name, seen
	return notices
}

// body is the notification text; with one proposal it names its first
// endpoint.
func (n notice) body() string {
	if len(n.Proposals) > 1 {
		return fmt.Sprintf("%d new policy proposals", len(n.Proposals))
	}
	if eps := n.Proposals[0].Endpoints; len(eps) > 0 {
		return "New policy proposal: " + eps[0]
	}
	return "New policy proposal"
}

// notifier sends notices as macOS notifications. It starts the Wails
// notification service itself on the first notice rather than as an app
// service: then macOS asks for permission only when there is something to
// show, and a failure is logged instead of stopping the app.
type notifier struct {
	mu      sync.Mutex
	service *notifications.NotificationService // nil until started
}

// send shows one notification per notice. It blocks while macOS asks for
// permission, so call it in its own goroutine.
func (n *notifier) send(notices []notice) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.service == nil {
		svc := notifications.New()
		if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
			log.Printf("notifications: %v", err)
			return
		}
		n.service = svc
	}
	// Asks only the first time; afterwards macOS answers right away.
	allowed, err := n.service.RequestNotificationAuthorization()
	if err != nil {
		log.Printf("notifications: %v", err)
		return
	}
	if !allowed {
		log.Print("notifications: not allowed in System Settings")
		return
	}
	for _, no := range notices {
		err := n.service.SendNotification(notifications.NotificationOptions{
			ID:    "proposal-" + no.Proposals[0].ID, // chunk IDs are unique
			Title: no.Sandbox,
			Body:  no.body(),
		})
		if err != nil {
			log.Printf("notifications: %v", err)
		}
	}
}
