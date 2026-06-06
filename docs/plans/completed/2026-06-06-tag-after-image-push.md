---
title: Tag Releases After Successful Image Push
status: completed
review_status: approved
template_version: 2.11.0
skill_version: 1.12.0
created_at: 2026-06-06T08:49:08Z
---

# Tag Releases After Successful Image Push

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.
Treat section ownership as fail-closed: `Progress` owns factual state transitions and stop points, `Surprises & Discoveries` owns still-active findings, `Decision Log` owns durable reusable decisions, and `Outcomes & Retrospective` owns milestone-level outcomes and remaining gaps. `Concrete Steps` owns planned commands, `Validation and Acceptance` owns planned proof, and `Implementation Completion Report` owns completed validation results.
Keep the title in this header and the YAML front matter in sync.
All recorded datetimes in this document must use UTC timestamps in the form `YYYY-MM-DDTHH:MM:SSZ`. Do not use date-only values.
Use `status` to track lifecycle progress. Allowed values are `planning`, `in_progress`, `completed`, and `failed`.
Use `review_status` to track human review state. Allowed values are `none` and `approved`.
Set `status: planning` while authoring or revising the ExecPlan before implementation starts, `status: in_progress` once implementation begins, `status: completed` only after human review confirms that all planned work and validation are finished, and `status: failed` only after human confirmation that the work is stopped due to failure or explicit cancellation.
Set `review_status: none` while authoring, implementing, or waiting for human review. Set `review_status: approved` only after a human explicitly approves the current outcome. Implementation-complete but review-pending work should remain `status: in_progress` with `review_status: none`.
No parent Strategy exists. Strategy was considered and is not required because this work fits one bounded workstream: move Git release marker creation behind successful image publishing in the existing tag-release workflow.
Before implementation starts, this ExecPlan explicitly names its primary deliverables as concrete repository changes.

## Purpose / Big Picture

After this change, a failed Docker build, Docker Hub push, or image signing step will not leave a Git release tag behind. A release tag such as `v0.4.17` will be pushed only after the Docker image has been pushed and signed successfully, so rerunning a failed release will not require manual tag deletion.

## Primary Deliverables

- Update `.github/workflows/tag-release.yml` so release tag creation runs after successful Docker image push and cosign signing.
- Update `hack/tag-release.sh` or add narrowly scoped scripts under `hack/` so version validation can run before the build without creating a Git tag.
- Preserve the existing `release-X.Y` branch creation behavior for `.0-beta.1` versions, but move it to the post-success release finalization path.

## Tracked Decision Record

No tracked decision record required.

## External API Parameters and Capability Checks

No external API used by repository code at runtime.

Workflow execution uses GitHub Actions, Docker Hub, and Sigstore cosign as existing release integrations. This plan does not add a new external integration or dependency. Capability checks before implementation:

- [x] Confirm `.github/workflows/tag-release.yml` currently owns release image push, signing, and tag finalization ordering.
- [x] Confirm `hack/tag-release.sh` currently creates and pushes `v<VERSION>` before the Docker image push.
- [x] Confirm `hack/get-version.sh` can provide the release version without creating tags.

Capability Check Status: PASS.
Checked at: 2026-06-06T09:03:50Z.
Evidence: `.github/workflows/tag-release.yml` runs `hack/tag-release.sh` before Docker login, buildx setup, `docker/build-push-action@v6`, and cosign signing. `hack/tag-release.sh` validates `RELEASE_VERSION`, creates annotated tag `v<VERSION>`, pushes it to `origin`, and creates the `release-X.Y` branch for `.0-beta.1` versions. `hack/get-version.sh` only reads and prints `RELEASE_VERSION`; it does not create tags or branches. Follow-up inspection found no downstream workflow step that depends on the Git tag already existing: image tagging uses `steps.version.outputs.version`, `GITVERSION` uses `${{ github.sha }}`, Dockerfile consumes `version.go` plus the `GITVERSION` build arg, and cosign signs the image ref and build digest.

## Implementation Gate (Hard Stop)

Plan-document creation and revision are allowed as plan-only work before implementation approval. No implementation-target changes are allowed until the required unlock checklist is complete, all items in `External API Parameters and Capability Checks` are completed and marked checked, and explicit user approval is recorded.

Blocked implementation-target actions before gate pass:
- `apply_patch` against `.github/workflows/tag-release.yml`
- `apply_patch` against `hack/tag-release.sh`
- adding or editing implementation scripts under `hack/`
- running code generation or formatting commands that modify implementation-target files

Required unlock checklist:
- [x] Capability checks are complete and recorded as `Capability Check Status: PASS` with `Checked at` and `Evidence`.
- [x] `Validation and Acceptance` includes acceptance criteria derived from user requirements, primary deliverables, and documented repository constraints.
- [x] Each non-obvious acceptance criterion maps back to its source, or the plan explains why a separate mapping list is unnecessary for this bounded workstream.
- [x] Explicit user approval is recorded as `User approval quote: "<exact quote>" (<UTC timestamp>)`.

User approval quote: "それでは実行を承認します。実行してください。" (2026-06-06T09:11:46Z)

## Progress

- [x] (2026-06-06T08:49:08Z) Created an ExecPlan for moving Git tag creation after successful image push and signing.
- [x] (2026-06-06T09:03:50Z) Completed capability checks and verified downstream release steps do not require the Git tag to exist before image build, push, or signing.
- [x] (2026-06-06T09:11:46Z) Recorded explicit implementation approval and opened the implementation gate.
- [x] (2026-06-06T09:14:39Z) Implemented release preflight and finalization ordering.
- [x] (2026-06-06T09:14:39Z) Ran validation commands: `bash -n hack/tag-release.sh`, `make test`, and `hack/tag-release.sh preflight`.
- [x] (2026-06-06T10:49:15Z) Human review completed and the plan was approved for closure.

## Surprises & Discoveries

- Observation: `docs/plans` did not exist before this plan request.
  Evidence: `ls docs/plans` failed before the directory and plan were created.

## Decision Log

- Decision: Treat cosign signing as part of release completion, so Git tag creation should happen after signing succeeds.
  Rationale: The current workflow runs signing after image push and marks signing failure as release failure; the Git release marker should represent the completed release artifact.
  Timestamp: 2026-06-06T08:49:08Z

- Decision: Treat an existing `v<VERSION>` tag as a preflight failure.
  Rationale: Reusing an already-tagged version would allow the workflow to build, push, and sign an image for a release marker that already exists, which is not a valid new release attempt.
  Timestamp: 2026-06-06T09:04:45Z

- Decision: Detect duplicate release tags from remote `origin` with `git ls-remote --tags origin "refs/tags/v${VERSION}"`.
  Rationale: The release invariant depends on whether the remote repository already has the release tag, not on which tags happen to be present in the local GitHub Actions checkout.
  Timestamp: 2026-06-06T09:06:06Z

## Outcomes & Retrospective

Implementation is complete and human review approved closure. The workflow now validates the release version and duplicate remote tag state before Docker image build, and creates Git release markers only after Docker image push and cosign signing have completed successfully.

## Context and Orientation

The release workflow is defined in `.github/workflows/tag-release.yml`. It runs on pushes to `main` and `release-*` branches when `version.go` changes.

The current workflow calls `hack/tag-release.sh` before Docker build and push. That script reads `RELEASE_VERSION` from `version.go`, validates it, creates the annotated Git tag `v<VERSION>`, and pushes that tag to `origin`. For versions ending in `.0-beta.1`, it also creates and pushes the corresponding `release-X.Y` branch.

The current Docker image publish step uses `docker/build-push-action@v6` with:

- `push: true`
- `platforms: linux/arm64,linux/amd64`
- `tags: norseto/taint-remover:v${{ steps.version.outputs.version }}`
- `provenance: true`
- `sbom: true`

The current signing step uses cosign to sign `norseto/taint-remover:v<VERSION>@<digest>`.

Non-obvious term: "release marker" means the Git tag and release branch that tell users and automation that a release exists.

## Observable Behavior and Compatibility

The intended observable behavior changes only for failed releases. When Docker build, Docker Hub push, or cosign signing fails, no Git release tag or release branch should be pushed.

Successful releases should keep the existing external behavior: the Docker image tag, cosign signature, Git tag, and beta release branch rule should still exist after the workflow succeeds.

No user-facing API or controller behavior changes are in scope.

## Lifecycle and Consistency Semantics

This work changes release lifecycle ordering. The invariant is that Git release markers are created only after release artifacts are published and signed.

The workflow should fail immediately with a non-zero exit when a duplicate Git tag already exists, before building or pushing an image. Duplicate tag detection must query remote `origin` with `git ls-remote --tags origin "refs/tags/v${VERSION}"` instead of relying on local checkout tag state. This intentionally changes the current `hack/tag-release.sh` behavior where an existing local tag match exits successfully. If the image push or signing fails, later Git tag and release branch creation must be skipped by normal GitHub Actions step ordering.

If Git tag creation fails after image push and signing have succeeded, the Docker image may remain published without a Git tag. Cleanup of already-pushed Docker images is out of scope for this bounded workstream.

## Plan of Work

First, inspect the current scripts and workflow to confirm the exact responsibilities and available version output behavior.

Then, separate release preflight from release finalization. The preflight path should validate `RELEASE_VERSION`, fail early on a remote duplicate `v<VERSION>` detected with `git ls-remote --tags origin "refs/tags/v${VERSION}"`, and expose the version to the workflow without creating tags or branches. The finalization path should create and push the Git tag, then preserve the existing `.0-beta.1` release branch behavior.

Finally, update `.github/workflows/tag-release.yml` so preflight runs before Docker build, and finalization runs only after Docker push and cosign signing complete successfully.

## Concrete Steps

Run these commands from `/workspaces/taint-remover`:

1. Inspect current release workflow and scripts.

       sed -n '1,240p' .github/workflows/tag-release.yml
       sed -n '1,240p' hack/tag-release.sh
       sed -n '1,120p' hack/get-version.sh

   Expected result: the workflow currently calls `hack/tag-release.sh` before Docker build, and `hack/tag-release.sh` creates and pushes the tag.

2. Edit release scripts.

   Target path: `hack/tag-release.sh` or new narrowly scoped files under `hack/`.

   Intended change: add a preflight mode or script that validates `version.go` and duplicate tag state without creating Git tags or branches. Duplicate tag detection must use `git ls-remote --tags origin "refs/tags/v${VERSION}"`. The preflight path must return a non-zero exit when `v<VERSION>` already exists on remote `origin`. Add a finalization mode or script that creates the tag and preserves the beta release branch behavior.

   Behavior to preserve: version format validation, duplicate tag detection, annotated tag message `Release <VERSION>`, and `.0-beta.1` release branch creation.

   Stop condition: if the current script behavior is relied on by another workflow or local command in a way that makes the split ambiguous, stop and ask for scope clarification.

3. Edit `.github/workflows/tag-release.yml`.

   Target path: `.github/workflows/tag-release.yml`.

   Intended change: replace the early tag creation step with preflight, keep Docker image tagging from the validated version output, and move release finalization after the cosign signing step.

   Behavior to preserve: Docker Hub login, buildx setup, multi-platform image build, provenance, SBOM, and cosign signing.

4. Run validation commands.

       git diff -- .github/workflows/tag-release.yml hack
       bash -n hack/tag-release.sh
       make test

   Expected result: shell syntax passes, tests pass, and the diff shows tag creation moved after image push and signing.

## Validation and Acceptance

### Requirement to Acceptance Mapping

- Source: User requirement that a failed image push must not require manual tag deletion.
  Acceptance: the workflow creates no Git tag before Docker build, Docker Hub push, and cosign signing have succeeded.
  Proof: inspect the workflow step order and confirm release finalization is after signing.

- Source: Primary deliverable to preserve release version behavior.
  Acceptance: version parsing and validation still reject invalid versions before the Docker build begins.
  Proof: run shell validation and, if practical, local dry-run or script-level checks for valid and invalid versions.

- Source: Lifecycle invariant that a release attempt must not reuse an existing Git release marker.
  Acceptance: duplicate `v<VERSION>` detection queries remote `origin` before Docker build and returns a non-zero exit when the remote tag exists.
  Proof: inspect preflight logic for `git ls-remote --tags origin "refs/tags/v${VERSION}"` and, if practical, run a script-level check with a remote tag fixture.

- Source: Existing repository behavior.
  Acceptance: `.0-beta.1` versions still create the matching `release-X.Y` branch, but only in finalization.
  Proof: inspect script logic and test or dry-run the branch condition if the implementation provides a safe dry-run path.

- Source: Current release workflow behavior.
  Acceptance: successful releases still publish `norseto/taint-remover:v<VERSION>` and sign the image before creating Git release markers.
  Proof: inspect workflow ordering and run repository validation commands.

Exact planned commands:

    bash -n hack/tag-release.sh
    make test

If new shell test files are added, run their exact test command as part of implementation completion. If Docker Hub push behavior cannot be fully tested locally because it requires release credentials, validate the workflow ordering by static inspection and explain that limitation in the completion report.

## Testing Strategy

Use shell syntax validation for release scripts and repository tests for general regression coverage.

If the implementation adds a dry-run mode or testable helper script, add focused shell tests for:

- duplicate tag detection before image build
- valid version extraction without tag creation
- finalization creating the release branch only for `.0-beta.1`

End-to-end GitHub Actions execution with Docker Hub credentials is not required before merging because it depends on release secrets and remote registry state. The workflow ordering must still be validated by review.

## Naming and Semantic Changes

No singular-to-collection naming changes are planned.

## Idempotence and Recovery

The preflight step should be safe to repeat because it must not create tags or branches.

The finalization step is not fully idempotent because pushing a Git tag is a persistent remote change. It should fail clearly if `v<VERSION>` already exists. If finalization fails after image push and signing, the Docker image may remain published without a Git tag; cleanup is out of scope for this workstream.

The key recovery improvement is that failures before finalization no longer require deleting Git tags.

## Artifacts and Notes

Current failure pattern being addressed:

    buildx failed with: unknown: blob unknown to registry

Current high-level ordering:

    tag-release.sh -> docker/build-push-action -> cosign sign

Target high-level ordering:

    release preflight -> docker/build-push-action -> cosign sign -> release finalization

## Interfaces and Dependencies

The workflow must continue using:

- `actions/checkout@v4`
- `docker/login-action@v3`
- `docker/setup-buildx-action@v3`
- `docker/build-push-action@v6`
- `sigstore/cosign-installer@v3`

Script interfaces must expose a validated release version to the workflow through `GITHUB_OUTPUT` or existing `hack/get-version.sh` behavior. Finalization must accept the version from `version.go` or from the same validated source and must create `v<VERSION>`.

## Implementation Completion Report

Original failure condition: the previous workflow created and pushed the Git release tag before Docker image build, Docker Hub push, and cosign signing. A later image push failure could leave a Git tag behind and require manual deletion before retry.

Fix made: `.github/workflows/tag-release.yml` now runs `hack/tag-release.sh preflight` before Docker login and build, uses the validated version output for the Docker image tag, and runs `hack/tag-release.sh finalize` only after cosign signing succeeds. `hack/tag-release.sh` now has explicit `preflight` and `finalize` modes. `preflight` validates `RELEASE_VERSION`, checks remote `origin` for `refs/tags/v<VERSION>` with `git ls-remote --tags`, returns non-zero for duplicate tags, and does not create tags or branches. `finalize` creates and pushes the annotated release tag, then preserves the `.0-beta.1` release branch behavior.

Regression tests or evidence added: no new test files were added. Validation evidence is `bash -n hack/tag-release.sh` passing, `make test` passing, static inspection of workflow ordering, and `hack/tag-release.sh preflight` returning non-zero with `Tag v0.5.0-alpha.7 already exists` for the current remote duplicate tag.

Reviewed behaviorally complete scope: `.github/workflows/tag-release.yml`, `hack/tag-release.sh`, `hack/get-version.sh`, Docker build inputs, Dockerfile `GITVERSION` handling, and cosign signing inputs.

Invariants checked during review: no Git tag or release branch is created before Docker build, Docker push, and cosign signing; duplicate remote release tags fail before Docker build; version validation still happens before Docker build; successful finalization keeps annotated tag message `Release <VERSION>` and `.0-beta.1` release branch creation.

Original, remaining, or new findings: no actionable remaining or new findings in the reviewed scope. End-to-end GitHub Actions execution with Docker Hub and Sigstore credentials was not run locally.

Validation results: `bash -n hack/tag-release.sh` passed. `make test` passed. `hack/tag-release.sh preflight` correctly failed before creating any tag because remote `origin` already contains `v0.5.0-alpha.7`.

## Change Notes

Created this ExecPlan to replace the earlier non-template plan with the required `exec-plan` structure and lifecycle gate.
