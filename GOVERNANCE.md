# Project governance

## Roles

- **Maintainers**: responsible for roadmap curation, triage, and releases.
  Current maintainers:
  - Aviv Sinai (@avivsinai)
- **Reviewers**: contributors with merge rights on specific areas (commands,
  clients, docs). Reviewers can approve PRs but maintainers perform final merges.
- **Contributors**: anyone sending PRs, docs updates, or filing issues.

## Decision making

- Day-to-day decisions happen in issues, PRs, or discussions.
- Significant changes (CLI UX, API surface, security posture) require an
  accepted design proposal or RFC tracked in `docs/rfcs/` (future enhancement).
- Maintainers aim for consensus. If consensus cannot be reached, the maintainer
  quorum decides.

## Meetings

We operate asynchronously. Ad-hoc maintainer syncs occur as needed; notes are
published in `docs/notes/` when applicable.

## Becoming a maintainer

- Demonstrate sustained, high-quality contributions for at least two release
  cycles.
- Show ownership over an area (e.g., pipelines, config, docs).
- Request nomination via GitHub Discussions. Existing maintainers will vote and
  record the outcome in the discussion.

## Stepping down

Maintainers may step down at any time by notifying the team. Access will be
revoked and CODEOWNERS updated accordingly.
