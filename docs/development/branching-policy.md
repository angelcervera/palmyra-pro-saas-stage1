---
id: branching-policy
version: 1.0.0
lastUpdated: 2025-11-04
appliesTo:
  - whole-repo
relatedDocs:
  - docs/api.md
  - docs/api-server.md
  - docs/project-requirements-document.md
---

# Git Branching & Release Policy

This repository follows a trunk‑based workflow optimized for a contract‑first monorepo. Contracts under `contracts/` are the source of truth; generated artifacts under `generated/` are read‑only.

## TL;DR

- Keep `main` always releasable. Work in short‑lived branches. Merge small PRs quickly once checks are green.
- Contract changes are explicit and come with regenerated code in the same PR.

## Branch Strategy

- Trunk‑based with short‑lived branches.
- Rebase frequently; avoid long‑lived integration branches.

## Branch Types & Naming

- `feat/<domain>-<topic>` — new functionality (e.g., `feat/users-approval-flow`).
- `fix/<domain>-<issue>` — bug fix (e.g., `fix/schema-slug-collision`).
- `chore/<area>-<task>` — deps, CI, formatting, tooling.
- `hotfix/<area>-<issue>` — urgent production fix cut from the last tag.
- `release/<yymmdd>-<name>` — optional short‑lived stabilization branch for coordinated deploys.

## Merging

- Prefer squash‑merge with a clean, scoped subject (present tense, optionally prefixed by domain, e.g., `users: approve/reject endpoints`).
- Keep branches small; rebase frequently to avoid drift.

## Branch Lifecycle

- Start from `main` and create a short‑lived branch using the naming scheme above.
- Rebase onto `main` frequently to pick up regenerated contracts and other cross‑cutting changes.
- Keep branches focused on a single concern; open additional branches rather than piling unrelated work into one.

### Hotfix flow

- Branch from the latest release tag (`hotfix/<area>-<issue>`), apply the fix, verify, then merge back into `main` and tag a PATCH release.

## Cross-Branch Coordination

- For large efforts, consider a temporary `release/<yymmdd>-<name>` branch. Only cherry-pick vetted fixes into it; delete after release.

- Avoid nested feature branches. If two people collaborate on the same scope, share the same short-lived branch or use multiple feature branches merged sequentially into `main`.

### Contract-first changes

- Prefer a single branch that carries the OpenAPI contract updates, regenerated artifacts, and the backend/frontend implementation work together. This keeps `main` compiling and avoids dangling contract-only branches.

## Releases & Tags (SemVer)

- Use semantic version tags on `main`: `vMAJOR.MINOR.PATCH`.
- Tag when cutting a deployable build from `main`. Pre‑releases allowed, e.g., `v1.4.0-rc.1`.

When to bump:
- MAJOR (`vX.0.0`): Any breaking change to public behavior or contracts (e.g., removing/renaming endpoints or fields in existing versions; incompatible type changes; changed auth/role requirements that break existing clients). Prefer introducing `/api/v2/...` for breaking API changes.
- MINOR (`vX.Y.0`): Backward‑compatible features (new endpoints; new optional fields; additive query params; performance improvements; non‑breaking migrations).
- PATCH (`vX.Y.Z`): Backward‑compatible bug fixes, internal refactors, ops/config/doc updates with no externally observable behavior change.

Optional stabilization:
- Create a short‑lived `release/<yymmdd>-<name>` branch when coordinating larger drops; only cherry‑pick fixes into it, then tag and delete.

## Breaking API Changes

- Avoid breaking changes in `v1`. If unavoidable, stage them behind a new contract version and path (e.g., `/api/v2/...`). Regenerate, update domains/SDKs, ensure all gates are green, then merge and cut a new MAJOR tag.

## Commit & PR Hygiene

- Commit contracts + regenerated artifacts together; describe `go generate` and test/build results in the PR body.
- Keep PRs focused and small; include screenshots for FE when user‑visible behavior changes.
