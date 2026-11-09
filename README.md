# shellight

A menu bar companion for [NVIDIA OpenShell](https://github.com/NVIDIA/OpenShell):
see your gateway and sandboxes at a glance, get notified about policy
proposals and react to them, without replacing the `openshell` CLI.

**Status:** early development, macOS (Apple silicon) only.

## What it does

- **Menu bar:** gateway status, sandboxes with their phase, running port
  forwards.
- **Dashboard** (planned): sandbox details, policy proposals to approve or
  reject, services, providers, settings.
- **Notifications:** new policy proposals, bundled per sandbox; sandboxes
  that fail or disappear (planned).

## Requirements

An installed and configured `openshell` CLI. shellight always follows the
CLI's active gateway (auth mode mtls, plaintext or none).

## Build

Everything builds inside the devcontainer:

```sh
task check   # go vet and tests
task app     # build/bin/shellight.app, ad hoc signed
```

## Docs

- [GLOSSARY.md](GLOSSARY.md): domain terms
- [docs/adr/](docs/adr/): architecture decisions
