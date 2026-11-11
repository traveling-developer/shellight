---
name: product-vision
description: "What shellight is for, who uses it, its form and platform (grilling session 2026-11-08)"
metadata:
  node_type: memory
  type: project
  originSessionId: eb572603-07cb-46af-ae88-103357b7e230
  modified: 2026-11-11T09:36:55.426Z
---

Decided by the user in the grilling session on 2026-11-08:

- **Users:** developers in general who run OpenShell locally (not only the person building it; not teams on shared/remote gateways).
- **Core job:** watch and react: status at a glance, a hint on new policy proposals, small targeted actions. A companion to the openshell CLI, not a replacement.
- **Form:** the menu bar shows only the essentials; everything else goes into a dashboard window.
- **Platform:** macOS first, Linux/Windows later.
- **CLI:** a prerequisite; shellight follows the CLI's active gateway, with no onboarding, login or gateway switcher of its own.
- **Auth modes:** mtls, plaintext and none; oidc and cloudflare_jwt get a clear "not supported" message.
- **Actions:** approve/reject a policy proposal, open a port forward in the browser, stop a port forward, open a shell into a sandbox in a terminal, restart the gateway (brew services). Not now: start forward, suspend/resume, create/delete sandbox.
- **Menu vs dashboard:** menu = status dot, gateway line ("Gateway: name · auth mode", 2026-11-11), sandboxes with name + phase only, running forwards as own "Port forwards" section (2026-11-11), "Open dashboard…", Quit. Dashboard = sandbox details (workspace, created, policy version, reason), policy proposals with approve/reject, other actions. Policy proposals never show in the menu.
- **Notifications:** new policy proposals, and a sandbox entering phase Error or disappearing. Not for an unreachable gateway (red dot is enough), not for proposals already open at app start. Click opens the dashboard at the affected item.
- **Distribution:** Homebrew cask in an own tap (`traveling-developer/homebrew-tap`), only from 0.1.0 (all actions done); until then local ad hoc builds.
- **Frontend:** Svelte 5 + TS, pinned exactly, Svelte scoped CSS + CSS variables (native look, light/dark), no Tailwind.
- **Design (adopted 2026-11-11):** "Native · Chartreuse" mockups in `docs/design/` are binding (values, not markup); their README lists the deviations. Menu, notifications and restart dialog stay native (NSMenu has no coloured/right-aligned text). Tray icon keeps its ">_" glyph, only colours change. Sidebar keeps the phase (the user rejected dropping it). Dashboard "Port forwards" section always shown, "none" when empty. Views the mockups lack are derived from their building blocks. Tickets 23 (look) and 24 (app icon) came out of this; each ticket has a "**Design:**" line.
- **Policy proposals:** dashboard shows the proposed rule (host:port, binary); approve/reject one at a time, no reject reason; no approve-all, edit or undo.
- **Confirmation:** only for gateway restart.
- **Gateway restart:** item only shown if the active gateway runs as a `brew services` service on this Mac; not on Linux/Windows for now.
- **Terminal:** configurable: a list of known installed terminals (Terminal.app, iTerm2, Ghostty — the maintainer uses Ghostty), default Terminal.app; custom command template only if needed later. Check how Ghostty launches a command on macOS when the step comes.
- **Notifications:** bundled per sandbox and poll round ("fwtest: 3 new policy proposals"; with one, the host is in the text).
- **Dock:** icon shows while the dashboard is open, hidden otherwise.
- **Settings:** a settings section in the dashboard, opened by the menu item "Settings…" and ⌘,. Settings so far: terminal, tray glyph colour (backlog), start at login (default off).
- **Dashboard layout:** sidebar of sandboxes grouped by workspace, badge = pending policy proposals, "Settings" at the bottom; detail pane for the selected sandbox.
- **Services:** read-only list per sandbox (name, URL), click opens the URL; no expose/delete.
- **Providers:** read-only. Workspace headings in the sidebar are clickable; the workspace view lists its providers (name, type, credential expiry). Sandbox details list the provider names it uses. Never show credentials or config values, not even masked. Expired or expiring (< 24 h) credentials are marked in the dashboard only (no notification, no dot change).
- **Version check:** gateway version vs the SDK version shellight is built with; hint if major or minor differ, patch ignored; CLI version not checked. Hint: "⚠ untested version" in the menu gateway line + banner with both versions in the dashboard; status dot unchanged.
- **Updates:** GitHub Releases API (traveling-developer/shellight, latest) at start and every 24 h; menu item "Update available: vX" opens the release page; settings show it with `brew upgrade --cask shellight`. Setting "Check for updates", default on; always off in dev builds. Only network access besides the gateway.
- **Language:** English only.
- **Build order (agreed 2026-11-08):** 1 rename Denial → PolicyProposal (while E1 is staged), 2 notification on new proposals, 3 dashboard skeleton + slim menu, 4 notification click opens dashboard, 5 approve/reject, 6 services, 7 stop forward, 8 settings + shell in terminal, 9 gateway restart, 10 notification on sandbox Error/gone, 11 start at login, 12 0.1.0 + Developer ID + notarization + cask. Providers, version check and update hint were added after this list; place them when we get there.
- **ADRs:** docs/adr/0001 (companion to the CLI), 0002 (proposals by polling the draft inbox).
- **Terms:** "policy proposal" = pending draft chunk (what the user reacts to); "denial" = the blocked event itself. See GLOSSARY.md.

**Why:** The domain model was unclear before; this is the shared baseline for all later steps.
**How to apply:** Check new features against "companion, not CLI replacement". Related: [[gateway-data-scope]], [[current-step]].
