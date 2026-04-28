#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

expected='system "#{bin}/bkt", "--version"'
old='system "#{bin}/bkt", "version"'

if grep -Fq "$old" .goreleaser.yaml; then
  fail ".goreleaser.yaml Homebrew test uses nonexistent 'bkt version' command"
fi

if ! grep -Fq "$expected" .goreleaser.yaml; then
  fail ".goreleaser.yaml Homebrew test must run 'bkt --version'"
fi
