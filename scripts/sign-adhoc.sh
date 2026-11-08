#!/usr/bin/env bash
# Ad hoc sign build/bin/shellight.app with rcodesign (no Apple identity).
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
app="$root/build/bin/shellight.app"

[[ -d "$app" ]] || { echo "$app missing, run: task build:darwin" >&2; exit 1; }

# rcodesign signs ad hoc when no certificate is given.
rcodesign sign "$app"

# `rcodesign verify` rejects ad hoc signatures (no CMS blob), so check the
# fields that matter instead: ad hoc flag, identifier, sealed Info.plist and
# resources.
info="$(rcodesign print-signature-info "$app/Contents/MacOS/shellight")"
for want in 'flags: CodeSignatureFlags(ADHOC)' 'identifier: dev.shellight.app' "'Info (1):" "'Resources (3):"; do
  grep -qF -- "$want" <<<"$info" || { echo "signature check failed: $want" >&2; exit 1; }
done
echo "==> ad hoc signature ok"
