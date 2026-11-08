# shellight

shellight is a desktop companion to the NVIDIA OpenShell CLI: it shows what
the active gateway and its sandboxes are doing and lets the user react,
without replacing the CLI.

## Language

**Active gateway**:
The OpenShell gateway the openshell CLI currently points at; shellight always
follows it and never picks a gateway of its own.
_Avoid_: current gateway, selected gateway, server

**Workspace**:
A named group on a gateway that holds sandboxes and providers.
_Avoid_: namespace, project

**Sandbox**:
An isolated environment on the gateway in which an agent runs under a policy.
_Avoid_: container, VM, box

**Provider**:
A registered AI provider with its credentials, belonging to a workspace;
sandboxes refer to providers by name.
_Avoid_: credential, integration, account

**Service**:
An HTTP endpoint of a sandbox that the gateway exposes under a URL.
_Avoid_: port forward, endpoint, route

**Dashboard**:
shellight's window with the details of every sandbox and the actions on them;
the menu bar shows only the essentials.
_Avoid_: main window, detail window, UI

**Port forward**:
A local port that the openshell CLI tunnels into a sandbox; it belongs to the
CLI, not to the gateway.
_Avoid_: tunnel, service

**Policy proposal**:
A pending entry in a sandbox's policy draft inbox that suggests allowing
something the sandbox's policy blocked; the user approves or rejects it.
_Avoid_: denial, draft chunk, request

**Denial**:
A single connection or action that a sandbox's policy blocked, as an event at
a point in time.
_Avoid_: block, violation, policy proposal
