---
name: gateway-data-scope
description: What shellight reads from the OpenShell gateway and via which source (decided 2026-11-08)
metadata:
  type: project
---

Decided by Christoph on 2026-11-08:

- **Source:** gRPC through NVIDIA's Go SDK, not the openshell CLI. Exception: port forwards are local CLI state (`openshell forward`), so listing them stays a CLI job.
- **Gateway:** reachable yes/no, version, compute driver (`GetGatewayInfo`). Auth status was not chosen.
- **Sandboxes (all workspaces):** name + phase, failure reason from `conditions`, workspace, created time, policy version.
- **Later steps (one at a time):** port forwards, policy denials, services, providers. Check how the SDK covers each when its step comes up.
- **Polling:** every 5 s, back off to 30 s on errors, refresh on menu open if Wails allows.

**Why:** Christoph picked these from the full list of what the API and CLI offer.
**How to apply:** Build only these fields; ask before adding others. Related: [[ask-scope-not-tmp-old]], [[current-step]].
