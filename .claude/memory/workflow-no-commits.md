---
name: workflow-no-commits
description: Rebuild of shellight from tmp_old in small steps; Claude stages and proposes a commit message but never commits
metadata:
  type: feedback
---

The project is rebuilt step by step from `tmp_old/` (old spike code, plan in `tmp_old/docs/mvp-tasks.md`). One small step at a time. Claude never runs `git commit`: it stages the files and proposes a commit message; Christoph commits.

**Why:** Christoph wants to review and commit each step themselves (stated 2026-11-08).
**How to apply:** End each step with staged changes, a commit message proposal and a suggestion for the next step. Related: [[lowercase-app-name]].
