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

codesign --force --sign - --identifier "$identifier" "$binary_path"
