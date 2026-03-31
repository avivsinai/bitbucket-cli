#!/usr/bin/env bash
# Bumps all skill + plugin manifest versions and creates a release tag.
# The tag push triggers both the binary release and skill publish workflows.
#
# Usage: ./scripts/release-skills.sh X.Y.Z
set -euo pipefail

version="${1:?usage: ./scripts/release-skills.sh X.Y.Z}"
tag="v${version}"

printf '%s' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$' || {
  echo "error: version must be X.Y.Z or X.Y.Z-rc.1, got: $version" >&2
  exit 1
}

git diff --quiet --ignore-submodules HEAD -- || {
  echo "error: worktree is dirty; commit or stash first" >&2
  exit 1
}
git diff --cached --quiet --ignore-submodules -- || {
  echo "error: index has staged changes; commit or unstage first" >&2
  exit 1
}

git fetch --tags --quiet origin 2>/dev/null || true
if git rev-parse "$tag" >/dev/null 2>&1; then
  echo "error: tag $tag already exists" >&2
  exit 1
fi

python3 - "$version" <<'PY'
import json, pathlib, re, sys

version = sys.argv[1]
changed = []

for path in sorted(pathlib.Path("skills").glob("*/SKILL.md")):
    text = path.read_text()
    m = re.match(r"(?s)^(---\n)(.*?)(\n---\n.*)$", text)
    if not m:
        print(f"  skip: {path} (no frontmatter)")
        continue
    head, fm, tail = m.groups()
    if not re.search(r"(?m)^version:\s*.+$", fm):
        print(f"  skip: {path} (no version field)")
        continue
    fm_new = re.sub(r"(?m)^version:\s*.+$", f"version: {version}", fm, count=1)
    if fm_new != fm:
        path.write_text(head + fm_new + tail)
        changed.append(str(path))

for p in [".claude-plugin/plugin.json", ".codex-plugin/plugin.json"]:
    pp = pathlib.Path(p)
    if pp.exists():
        data = json.loads(pp.read_text())
        if data.get("version") != version:
            data["version"] = version
            pp.write_text(json.dumps(data, indent=2) + "\n")
            changed.append(str(pp))

for f in changed:
    print(f"  bumped: {f}")
if not changed:
    print("  (no files needed bumping)")
PY

git add skills/*/SKILL.md .claude-plugin/plugin.json
[ -f .codex-plugin/plugin.json ] && git add .codex-plugin/plugin.json

./scripts/check-release-version.sh "$version"

if git diff --cached --quiet; then
  echo "No version changes needed — tagging current HEAD"
else
  git commit -m "chore: release skills ${tag}"
fi

git tag -a "$tag" -m "Release ${tag}"

echo ""
echo "Created tag ${tag}"
echo "Next:"
echo "  git push origin HEAD && git push origin ${tag}"
