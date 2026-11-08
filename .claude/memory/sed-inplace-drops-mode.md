---
name: sed-inplace-drops-mode
description: "In this devcontainer sandbox, sed -i resets file mode to 0600 (exec bit lost on scripts)"
metadata:
  node_type: memory
  type: project
  originSessionId: d62ca75d-3303-47ad-b11c-96fc64c24e67
  modified: 2026-11-08T12:50:48.283Z
---

`sed -i` on a file in the repo left it with mode 0600 (seen 2026-11-08 on `scripts/sign-adhoc.sh` and `README.md`); `task app` then failed with "permission denied".

**Why:** The sandbox rewrites the file instead of keeping its mode; git shows it as a mode change.
**How to apply:** Edit files with the Edit tool, or check `git diff --summary` / `ls -l` after shell edits and restore the mode (0755 for scripts, 0644 otherwise).
