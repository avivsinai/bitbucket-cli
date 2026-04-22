#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/release.sh X.Y.Z [options]

Prepares a release PR from master by:
- verifying the worktree is clean
- verifying HEAD matches origin/master
- creating release/vX.Y.Z
- moving CHANGELOG.md's Unreleased section into a versioned release entry
- bumping skill/plugin metadata versions
- validating the release metadata
- optionally running check-skills/gofmt/vet/test/build
- creating and pushing the release commit
- opening a GitHub PR and enabling squash auto-merge

Options:
  --date YYYY-MM-DD  Override release date (default: today in UTC)
  --skip-verify      Skip local verification gates
  --allow-empty      Allow releasing with an empty Unreleased section
  --no-auto-merge    Create the PR but do not enable auto-merge
  -h, --help         Show this help text

Examples:
  ./scripts/release.sh 1.2.3
  ./scripts/release.sh 1.2.3 --no-auto-merge
EOF
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

version=""
release_date="$(date -u +%Y-%m-%d)"
skip_verify=0
allow_empty=0
auto_merge=1

while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --date)
      [ $# -ge 2 ] || { echo "error: --date requires a value" >&2; exit 1; }
      release_date="$2"
      shift 2
      ;;
    --skip-verify)
      skip_verify=1
      shift
      ;;
    --allow-empty)
      allow_empty=1
      shift
      ;;
    --no-auto-merge)
      auto_merge=0
      shift
      ;;
    --*)
      echo "error: unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
    *)
      if [ -n "$version" ]; then
        echo "error: version already set to $version; unexpected extra argument: $1" >&2
        usage >&2
        exit 1
      fi
      version="$1"
      shift
      ;;
  esac
done

[ -n "$version" ] || { usage >&2; exit 1; }

require_command git
require_command gh
require_command python3

printf '%s' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$' || {
  echo "error: version must be X.Y.Z or X.Y.Z-rc.1, got: $version" >&2
  exit 1
}
printf '%s' "$release_date" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' || {
  echo "error: date must be YYYY-MM-DD, got: $release_date" >&2
  exit 1
}

tag="v${version}"
branch="release/${tag}"

git diff --quiet --ignore-submodules HEAD -- || {
  echo "error: worktree is dirty; commit or stash first" >&2
  exit 1
}
git diff --cached --quiet --ignore-submodules -- || {
  echo "error: index has staged changes; commit or unstage first" >&2
  exit 1
}

current_branch="$(git branch --show-current)"
[ "$current_branch" = "master" ] || {
  echo "error: releases must be prepared from master; current branch is $current_branch" >&2
  exit 1
}

git fetch --quiet origin master --tags

local_head="$(git rev-parse HEAD)"
remote_head="$(git rev-parse origin/master)"
[ "$local_head" = "$remote_head" ] || {
  echo "error: local master is not at origin/master; pull or reset before releasing" >&2
  exit 1
}

if git show-ref --verify --quiet "refs/heads/${branch}"; then
  echo "error: branch already exists locally: ${branch}" >&2
  exit 1
fi
if git ls-remote --exit-code --heads origin "${branch}" >/dev/null 2>&1; then
  echo "error: branch already exists on origin: ${branch}" >&2
  exit 1
fi
if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null 2>&1; then
  echo "error: tag already exists locally: ${tag}" >&2
  exit 1
fi
if git ls-remote --exit-code --tags origin "refs/tags/${tag}" >/dev/null 2>&1; then
  echo "error: tag already exists on origin: ${tag}" >&2
  exit 1
fi

git switch -c "$branch"

python3 - "$version" "$release_date" "$allow_empty" <<'PY'
import json
import pathlib
import re
import sys

version, release_date, allow_empty = sys.argv[1], sys.argv[2], sys.argv[3] == "1"
repo_compare_prefix = "https://github.com/avivsinai/bitbucket-cli/compare/"
repo_release_prefix = "https://github.com/avivsinai/bitbucket-cli/releases/tag/"

changelog = pathlib.Path("CHANGELOG.md")
text = changelog.read_text()
marker = "## [Unreleased]"
if marker not in text:
    raise SystemExit("error: CHANGELOG.md is missing the Unreleased section")

start = text.index(marker)
after_marker = start + len(marker)
rest = text[after_marker:]
match = re.search(r"(?m)^## \[", rest)
if match:
    unreleased_body = rest[:match.start()]
    suffix = rest[match.start():]
else:
    unreleased_body = rest
    suffix = ""

if not unreleased_body.strip() and not allow_empty:
    raise SystemExit("error: CHANGELOG.md Unreleased section is empty; add release notes first or pass --allow-empty")

release_header = f"\n\n## [{version}] - {release_date}\n"
new_text = text[:start] + marker + release_header + unreleased_body.lstrip("\n")
if suffix:
    new_text += suffix if suffix.startswith("\n") else "\n" + suffix

versions = re.findall(r"(?m)^## \[([0-9A-Za-z.+-]+)\] - \d{4}-\d{2}-\d{2}$", new_text)
if not versions:
    raise SystemExit("error: CHANGELOG.md contains no version headings")

footer_lines = [f"[Unreleased]: {repo_compare_prefix}v{versions[0]}...HEAD"]
for i, current in enumerate(versions):
    if i + 1 < len(versions):
        footer_lines.append(
            f"[{current}]: {repo_compare_prefix}v{versions[i + 1]}...v{current}"
        )
    else:
        footer_lines.append(f"[{current}]: {repo_release_prefix}v{current}")

new_text = re.sub(
    r"(?ms)\n\[Unreleased\]: https://github\.com/avivsinai/bitbucket-cli/.*\Z",
    "",
    new_text.rstrip(),
)
new_text = new_text.rstrip() + "\n\n" + "\n".join(footer_lines) + "\n"
changelog.write_text(new_text)

changed = ["CHANGELOG.md"]

for path in sorted(pathlib.Path("skills").glob("*/SKILL.md")):
    skill_text = path.read_text()
    m = re.match(r"(?s)^(---\n)(.*?)(\n---\n.*)$", skill_text)
    if not m:
        raise SystemExit(f"error: {path} is missing YAML frontmatter")
    head, fm, tail = m.groups()
    if not re.search(r"(?m)^version:\s*.+$", fm):
        raise SystemExit(f"error: {path} is missing a version field")
    fm_new = re.sub(r"(?m)^version:\s*.+$", f"version: {version}", fm, count=1)
    if fm_new != fm:
        path.write_text(head + fm_new + tail)
        changed.append(str(path))

for path in [".claude-plugin/plugin.json", ".codex-plugin/plugin.json"]:
    pp = pathlib.Path(path)
    if not pp.exists():
        continue
    data = json.loads(pp.read_text())
    if data.get("version") != version:
        data["version"] = version
        pp.write_text(json.dumps(data, indent=2) + "\n")
        changed.append(path)

print("prepared release files:")
for path in changed:
    print(f"  - {path}")
PY

./scripts/check-release-version.sh "$tag"

if [ "$skip_verify" -eq 0 ]; then
  make check-skills
  files="$(gofmt -l .)"
  if [[ -n "$files" ]]; then
    echo "error: gofmt wants changes:" >&2
    echo "$files" >&2
    exit 1
  fi
  go vet ./...
  make test
  make build
fi

git add CHANGELOG.md skills/*/SKILL.md .claude-plugin/plugin.json
[ -f .codex-plugin/plugin.json ] && git add .codex-plugin/plugin.json

git diff --cached --quiet && {
  echo "error: release prep produced no staged changes" >&2
  exit 1
}

git commit -m "chore(release): ${tag}"
git push -u origin "$branch"

pr_body=$(
  cat <<EOF
## Release

- moves \`CHANGELOG.md\` into \`${tag}\`
- aligns skill/plugin metadata to \`${version}\`
- merge triggers \`.github/workflows/release.yml\`, which validates the release commit, creates \`${tag}\`, publishes the release, and publishes skills in the same workflow run
EOF
)

pr_url="$(
  gh pr create \
    --base master \
    --head "$branch" \
    --title "chore(release): ${tag}" \
    --body "$pr_body"
)"

if [ "$auto_merge" -eq 1 ]; then
  gh pr merge --auto --squash --delete-branch "$pr_url" || {
    echo "error: failed to enable auto-merge; verify repository auto-merge support or rerun with --no-auto-merge" >&2
    exit 1
  }
fi

echo ""
echo "Prepared ${tag}"
echo "Release branch: ${branch}"
echo "Pull request: ${pr_url}"
if [ "$auto_merge" -eq 1 ]; then
  echo "Auto-merge: enabled (squash)"
else
  echo "Auto-merge: not enabled"
fi
