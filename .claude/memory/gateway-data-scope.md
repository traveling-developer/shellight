---
name: gateway-data-scope
description: What shellight reads from the OpenShell gateway and via which source (decided 2026-11-08)
metadata:
  node_type: memory
  type: project
  originSessionId: 3d2f5269-054f-4e48-ae3e-a2356fbe35b5
  modified: 2026-11-11T15:40:11.219Z
---

Decided by the user on 2026-11-08:

- **Source:** gRPC through NVIDIA's Go SDK, not the openshell CLI. Exception: port forwards are local CLI state (`openshell forward`), so listing them stays a CLI job.
- **Gateway:** reachable yes/no, version, compute driver (`GetGatewayInfo`). Auth status was not chosen.
- **Sandboxes (all workspaces):** name + phase, failure reason from `conditions`, workspace, created time, policy version.
- **Later steps (one at a time):** port forwards, policy proposals, services, providers. Check how the SDK covers each when its step comes up.
- **Polling:** every 5 s; after an error 6 more rounds at 5 s, then back off to 30 s (changed in ticket 27, confirmed 2026-11-11, so the menu can say "retrying every 5 s"). "Last seen" and the failure count belong to the active gateway and start over when it changes. No push for the sandbox list or gateway health exists; `WatchSandbox` streams only one named sandbox. No menu-will-open hook in Wails on macOS (only a custom OnClick, which loses native menu tracking).

- **Port forwards (decided 2026-11-08, step D):** run `openshell forward list -o json` each poll round (the SDK cannot list CLI forwards; it is local CLI state at `<config>/openshell/forwards/<ws>/<sandbox>-<port>.pid` and has no gateway field). Show running forwards only (dead ones hidden, review 2026-11-08). Since ticket 26 (2026-11-11) the menu lists them in its own "Port forwards" section as "host:port — sandbox", including forwards to sandboxes the active gateway does not list (user decision; they may belong to another gateway, as forwards have no gateway field); the dashboard shows only those of its sandboxes. One action: open http://host:port in the browser (localhost for loopback and wildcard binds). No stop action. Finder apps have no shell PATH, so the binary is looked up on PATH and then at /opt/homebrew/bin and /usr/local/bin.
- **Policy proposals (decided 2026-11-08, step E):** source is the draft inbox, `Policy().GetDraft` per sandbox each poll round (the SDK's `Sandboxes().Watch` only follows status, and the `DraftPolicyUpdate` push in the proto is not exposed by the SDK, so no per-sandbox streams). Show chunks with status pending only (server-side filter; rejected dropped in review 2026-11-08, "start with pending"); fields: chunk ID, endpoint (host:port) and binary. A failing GetDraft sets `Status.ProposalErr` like ForwardErr and does not fail the round. Not wanted: OCSF log lines, hit count, rationale, list in the menu, count in the sandbox line, status dot. Visibility: a macOS notification for new policy proposals, one per sandbox and poll round; clicking it opens a window that shows the proposal. the user: "nicht alles auf einmal", so split: E1 read the proposals (no UI), E2 notification, E3 window on click. Approve/reject is a later step.

**Why:** The user picked these from the full list of what the API and CLI offer.
**How to apply:** Build only these fields; ask before adding others. Related: [[current-step]], [[product-vision]].
