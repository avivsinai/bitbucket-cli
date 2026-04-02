#!/usr/bin/env bash
set -euo pipefail

raw_version="${1:?usage: ./scripts/check-release-version.sh [v]X.Y.Z}"
version="${raw_version#v}"

printf '%s' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$' || {
  echo "error: version must be X.Y.Z or X.Y.Z-rc.1, got: $raw_version" >&2
  exit 1
}

python3 - "$version" <<'PY'
import json, pathlib, re, sys

version = sys.argv[1]
mismatches = []

for path in sorted(pathlib.Path("skills").glob("*/SKILL.md")):
    text = path.read_text()
    match = re.match(r"(?s)^---\n(.*?)\n---\n", text)
    if not match:
        mismatches.append((str(path), "<missing frontmatter>"))
        continue
    frontmatter = match.group(1)
    version_match = re.search(r"(?m)^version:\s*([0-9A-Za-z.+-]+)\s*$", frontmatter)
    if not version_match:
        mismatches.append((str(path), "<missing version>"))
        continue
    actual = version_match.group(1)
    if actual != version:
        mismatches.append((str(path), actual))

for path in [".claude-plugin/plugin.json", ".codex-plugin/plugin.json"]:
    pp = pathlib.Path(path)
    if not pp.exists():
        continue
    actual = json.loads(pp.read_text()).get("version")
    if actual != version:
        mismatches.append((path, actual))

changelog = pathlib.Path("CHANGELOG.md")
if not changelog.exists():
    mismatches.append(("CHANGELOG.md", "<missing>"))
else:
    text = changelog.read_text()
    pattern = rf"(?m)^## \[{re.escape(version)}\] - \d{{4}}-\d{{2}}-\d{{2}}$"
    if not re.search(pattern, text):
        mismatches.append(("CHANGELOG.md", "<missing release heading>"))

if mismatches:
    print(f"release metadata version mismatch for {version}:")
    for path, actual in mismatches:
        print(f"  - {path}: {actual!r}")
    raise SystemExit(1)

print(f"release metadata matches {version}")
PY
