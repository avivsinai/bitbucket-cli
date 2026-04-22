#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/update-nix-vendor-hash.sh [flake-file] [build-target]

Refreshes buildGoModule's vendorHash by temporarily using pkgs.lib.fakeHash,
running a Nix build, and replacing the hash with the value reported by Nix.

Examples:
  ./scripts/update-nix-vendor-hash.sh
  ./scripts/update-nix-vendor-hash.sh flake.nix .#default
EOF
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

flake_file="${1:-flake.nix}"
build_target="${2:-.#default}"

case "${flake_file}" in
  -h|--help)
    usage
    exit 0
    ;;
esac

require_command nix
require_command python3

[ -f "$flake_file" ] || {
  echo "error: flake file not found: $flake_file" >&2
  exit 1
}

backup_file="$(mktemp)"
log_file="$(mktemp)"
keep_changes=0

cleanup() {
  if [ "$keep_changes" -eq 0 ] && [ -f "$backup_file" ]; then
    cp "$backup_file" "$flake_file"
  fi
  rm -f "$backup_file" "$log_file"
}

trap cleanup EXIT

cp "$flake_file" "$backup_file"

python3 - "$flake_file" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
text = path.read_text(encoding="utf-8")
pattern = re.compile(r'vendorHash\s*=\s*(?:"[^"]*"|pkgs\.lib\.fakeHash|lib\.fakeHash|null);')
replacement = "vendorHash = pkgs.lib.fakeHash;"
updated, count = pattern.subn(replacement, text, count=1)
if count != 1:
    raise SystemExit(f"error: expected exactly one vendorHash assignment in {path}")
path.write_text(updated, encoding="utf-8")
PY

if nix build "$build_target" --no-link --print-build-logs >"$log_file" 2>&1; then
  echo "error: nix build unexpectedly succeeded; vendorHash was not refreshed" >&2
  exit 1
fi

new_hash="$(
  sed -n 's/.*got:[[:space:]]*\(sha256-[A-Za-z0-9+/=]*\).*/\1/p' "$log_file" | tail -n 1
)"

[ -n "$new_hash" ] || {
  cat "$log_file" >&2
  echo "error: could not extract vendorHash from nix build output" >&2
  exit 1
}

python3 - "$flake_file" "$new_hash" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
new_hash = sys.argv[2]
text = path.read_text(encoding="utf-8")
pattern = re.compile(r'vendorHash\s*=\s*(?:"[^"]*"|pkgs\.lib\.fakeHash|lib\.fakeHash|null);')
replacement = f'vendorHash = "{new_hash}";'
updated, count = pattern.subn(replacement, text, count=1)
if count != 1:
    raise SystemExit(f"error: expected exactly one vendorHash assignment in {path}")
path.write_text(updated, encoding="utf-8")
PY

keep_changes=1
echo "Updated $flake_file vendorHash to $new_hash"
