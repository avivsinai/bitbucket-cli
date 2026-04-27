#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

if ! grep -Fq 'pkg/oauth.cloudClientID={{ .Env.BKT_OAUTH_CLIENT_ID }}' .goreleaser.yaml; then
  fail ".goreleaser.yaml must embed BKT_OAUTH_CLIENT_ID via ldflags"
fi
if ! grep -Fq 'pkg/oauth.cloudClientSecret={{ .Env.BKT_OAUTH_CLIENT_SECRET }}' .goreleaser.yaml; then # gitleaks:allow
  fail ".goreleaser.yaml must embed BKT_OAUTH_CLIENT_SECRET via ldflags"
fi

if ! grep -Fq 'BKT_OAUTH_CLIENT_ID: ${{ secrets.BKT_OAUTH_CLIENT_ID }}' .github/workflows/release.yml; then
  fail "release workflow must pass BKT_OAUTH_CLIENT_ID to release builds"
fi
if ! grep -Fq 'BKT_OAUTH_CLIENT_SECRET: ${{ secrets.BKT_OAUTH_CLIENT_SECRET }}' .github/workflows/release.yml; then
  fail "release workflow must pass BKT_OAUTH_CLIENT_SECRET to release builds"
fi
if ! grep -Fq 'BKT_OAUTH_CLIENT_ID and BKT_OAUTH_CLIENT_SECRET must be set as repository secrets' .github/workflows/release.yml; then
  fail "release workflow must validate OAuth release secrets before publishing"
fi

oauth_ldflags="-X github.com/avivsinai/bitbucket-cli/pkg/oauth.cloudClientID=build-id -X github.com/avivsinai/bitbucket-cli/pkg/oauth.cloudClientSecret=build-secret" # gitleaks:allow
go test -run 'TestCloudClient(ID|Secret)FromLdflags' \
  -ldflags "$oauth_ldflags" \
  ./pkg/oauth
