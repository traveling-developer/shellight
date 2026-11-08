---
name: ask-scope-not-tmp-old
description: "Ask Christoph what data/fields a step should read or expose; tmp_old is a loose reference, not the spec"
metadata:
  node_type: memory
  type: feedback
  originSessionId: 39540698-2dbe-4544-961b-645c29407094
  modified: 2026-11-08T13:16:57.333Z
---

Before building a step, ask Christoph which data we actually want to read or expose (fields, files, flags), instead of copying the scope of `tmp_old/` or its plan `tmp_old/docs/mvp-tasks.md`.

**Why:** In M1 I added the mTLS bundle paths (`CAFile`, `CertFile`, `KeyFile`), `TLSConfig`, typed errors and a `-gateway` flag because tmp_old/the old plan had them; Christoph removed most of it as not needed yet (2026-11-08).
**How to apply:** At the start of each step, list what I think is needed and ask (AskUserQuestion) before writing code; build only the minimum agreed. Treat tmp_old as background, not as a template. Related: [[workflow-no-commits]].
