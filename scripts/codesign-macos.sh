#!/usr/bin/env bash
set -euo pipefail

binary_path="${1:?usage: ./scripts/codesign-macos.sh /path/to/binary identifier [target-os]}"
identifier="${2:?usage: ./scripts/codesign-macos.sh /path/to/binary identifier [target-os]}"
target_os="${3:-darwin}"

if [[ "$target_os" != "darwin" ]]; then
  exit 0
fi

if [[ ! -f "$binary_path" ]]; then
  echo "error: binary not found: $binary_path" >&2
  exit 1
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
  exit 0
fi

# Sign ad-hoc with an explicit Designated Requirement pinned to the identifier.
#
# Without -r, codesign derives a DR of the form `cdhash H"..."` which changes
# on every rebuild. Each new bkt release would therefore invalidate any "Always
# Allow" the user granted in Keychain and re-prompt on the next invocation.
# Pinning the DR to the identifier makes the DR stable across rebuilds so the
# Keychain ACL keeps matching after `brew upgrade bkt`.
#
# Security trade-off: any other ad-hoc binary that claims this identifier
# satisfies the DR. Acceptable because the Keychain item only stores a user-
# scoped Bitbucket token, trust is granted per-host by the user explicitly via
# `bkt auth login`, and the identifier is namespaced under our GitHub org.
# Upgrading to a Developer ID signature (stricter anchor) is tracked as
# separate future work.
codesign \
  --force \
  --sign - \
  --identifier "$identifier" \
  -r="designated => identifier \"$identifier\"" \
  "$binary_path"
