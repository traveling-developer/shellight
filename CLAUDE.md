# shellight

A macOS menu bar companion for the NVIDIA OpenShell CLI (Go, Wails v3). Product
scope and terms: `GLOSSARY.md`, `docs/adr/`. Use the glossary's terms in code
and docs (e.g. "policy proposal", not "denial", for a pending draft chunk).

- The name is always lowercase: `shellight`, also in the bundle name, UI and docs.
- Code, comments, docs and commit messages in English; talk to the user in German.

## Workflow

- Work from the tickets in `.scratch/mvp/issues/`, one at a time. Start a
  ticket only when all its "Blocked by" tickets are done; set its status to
  `done` after the user has committed it.
- Before coding a step, list the data, fields and behaviour you think it needs
  and ask the user; build only the agreed minimum. `tmp_old/` (old spike) is
  background, not a template.
- Never run `git commit`. End each step with the files staged, a proposed
  commit message and a suggestion for the next step; the user commits.
- Ask before adding any Go or npm dependency.

## Safety

- Write actions (approve/reject, stop forward, restart, …) only against the
  test gateway in the devcontainer, never against a real gateway.
- Never open the real CLI config (`~/.config/openshell`); development uses the
  test config under `.dev/test-gateway/`.
- Never log process command lines: the ssh argv of port forwards carries a
  session token.

## Build and test

Everything runs inside the devcontainer; nothing runs on the Mac except the
built app.

- `task check`: go vet and all tests
- `task app`: cross build and ad hoc sign `build/bin/shellight.app` (darwin/arm64)
- `scripts/test-gateway.sh up | down | cli <openshell args>`: local test
  gateway (mTLS) on nested Podman; sandbox `fwtest` has a pending policy
  proposal. `up` also starts a second gateway without TLS (auth mode
  plaintext, no sandboxes, config under `.dev/test-gateway-plaintext/`)
- `SHELLIGHT_LIVE=1 go test ./internal/gateway -run Live -v`: live tests
  against both test gateways

Host check: the user quits the running shellight, double-clicks
`build/bin/shellight.app` on the Mac and reports back. The menu shows the
build ID, so say which build to expect.
