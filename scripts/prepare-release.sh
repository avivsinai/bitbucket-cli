#!/usr/bin/env bash
set -euo pipefail

echo "warning: scripts/prepare-release.sh is deprecated; use scripts/release.sh instead" >&2
exec ./scripts/release.sh "$@"
