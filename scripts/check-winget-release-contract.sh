#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

config=.goreleaser.yaml
workflow=.github/workflows/release.yml
expected_skip_upload='skip_upload: '\''{{ if eq .Env.WINGET_SKIP_UPLOAD "false" }}false{{ else }}true{{ end }}'\'''

if ! grep -Fq 'token: "{{ .Env.WINGET_GITHUB_TOKEN }}"' "$config"; then
  fail ".goreleaser.yaml must read the WinGet token only from WINGET_GITHUB_TOKEN"
fi
if ! grep -Fq "$expected_skip_upload" "$config"; then
  fail ".goreleaser.yaml must default WinGet uploads to skipped"
fi

# shellcheck disable=SC2016 # Match the literal GitHub Actions expression.
if [ "$(grep -Fc 'WINGET_GITHUB_TOKEN: ${{ secrets.WINGET_GITHUB_TOKEN }}' "$workflow")" -ne 2 ]; then
  fail "release workflow must bind WINGET_GITHUB_TOKEN from the repository secret exactly twice"
fi
if ! grep -Fq 'id: winget' "$workflow"; then
  fail "release workflow must expose the WinGet configuration result"
fi
# shellcheck disable=SC2016 # Match the literal workflow command.
if ! grep -Fq 'echo "skip_upload=true" >> "$GITHUB_OUTPUT"' "$workflow"; then
  fail "release workflow must skip WinGet upload when the secret is absent"
fi
if ! grep -Fq 'echo "::warning::WinGet publishing skipped:' "$workflow"; then
  fail "release workflow must warn when WinGet publishing is skipped"
fi
# shellcheck disable=SC2016 # Match the literal GitHub Actions expression.
if ! grep -Fq 'WINGET_SKIP_UPLOAD: ${{ steps.winget.outputs.skip_upload }}' "$workflow"; then
  fail "release workflow must pass the WinGet skip decision to GoReleaser"
fi

if grep -Fq 'WINGET_GITHUB_TOKEN must be set as a repository secret' "$workflow"; then
  fail "a missing WinGet token must not fail the release"
fi
if grep -Eq '^[[:space:]]*set[[:space:]]+(-[^[:space:]]*x|-o[[:space:]]+xtrace)([[:space:]]|$)' "$workflow"; then
  fail "release workflow must not enable shell xtrace"
fi
