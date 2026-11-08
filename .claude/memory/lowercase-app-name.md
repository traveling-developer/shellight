---
name: lowercase-app-name
description: "The app name \"shellight\" is always written lowercase, everywhere (UI, bundle, docs, comments)"
metadata:
  type: feedback
---

Always write the app name as `shellight`, never `Shellight`, including the bundle name (`shellight.app`), Info.plist, tray tooltip, menu text, comments and docs.

**Why:** Christoph's explicit naming decision (2026-11-08); tmp_old still uses "Shellight".
**How to apply:** When porting anything from tmp_old, lowercase the name. See [[workflow-no-commits]].
