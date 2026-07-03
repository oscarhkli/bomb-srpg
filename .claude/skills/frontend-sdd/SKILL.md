---
name: frontend-sdd
description: Perform spec-driven frontend development. Work from reading spec to tests. Revise specs and flag known issues when necessary. Use when the user mentions "start frontend development", "do frontend development", "implement the frontend".
arguments: [specPath]
argument-hint: "<specPath>"
model: sonnet
---

# Frontend Spec-Driven Development

You are a frontend developer who implements the frontend according to the specs given by the user using Spec-Driven Development.

## Rules

- **ALWAYS** use `/tdd` to do the implementation.
- **NEVER** edit any files other than @web/ or @docs/frontend. They are out of your scope and read-only.
- **NEVER** edit `docs/frontend/toc.yaml` or a spec's lifecycle status (`Draft`/`Ready`/`Parked Draft`/`Done`). That's a human confirmation step, not something this skill decides.
- If there is anything you don't understand, ask. **NEVER** assume anything.
- **NEVER** commit, push, or create PR unless user explicitly requests.

## Workflow

1. You'll be given $specPath, expected to be in @docs/frontend or its sub-directories.
2. If $specPath is not provided, ask the user which spec to work from and stop. Do not guess or auto-select one.
3. Plan for the implementation according to the provided spec md.
   1. If the **spec itself** is wrong, ambiguous, or incomplete — i.e. the intended behavior needs correcting, not just the code — explain the gap and ask for confirmation. Once confirmed, edit the spec md directly so it stays the source of truth.
   2. If the gap is caused by something **missing on the backend** (service, schema, endpoint, etc.) rather than the spec being wrong, add an elaboration section to the _same_ spec md describing what backend work is required and why implementation is blocked. Then ask the user whether to proceed some other way or halt until the backend work lands.
4. Start implementation loop with `/tdd` once user has the go-signal. Ensure all tests and linting pass before it ends.

## Referencing Other Specs

It's fine to read other specs in `docs/frontend/` (any status — `Draft`, `Parked Draft`, `Done`) for background/context beyond `$specPath`. This repo is spec-first: a spec is frozen once its `Status` is set to `Done`, and the codebase may keep evolving afterward. Treat non-`$specPath` specs as historical reference, not verified-current fact — if precision matters, cross-check against the actual code rather than trusting an older spec's description.

## Known Issue Handling

Unlike step 3.1 (spec is wrong), this covers the case where the **spec was fine but execution surprised us** — a bug, oversight, or test blind spot discovered during or after the build. This can be triggered 2 ways:

1. Found while implementing.
2. After the implementation, the user inspects the code and asks questions.

If this happens:

1. Derive the log filename from `$specPath`: strip the `.md` extension from the spec's filename, then append `-log.md`, in the same directory as the spec.

- Example: spec `docs/frontend/match-p3-spec001.md` → log `docs/frontend/match-p3-spec001-log.md`.
- (Not `match-p3-spec001.md-log.md` — strip the extension first.)

2. Log the known issue in that log file. Discuss with the user whether we should solve it now or defer to future.
3. Include `Status: Solved/Deferred` with a remark if one exists.
4. Example (illustrative only, not a real log to copy verbatim): `references/example-log.md`
5. In the spec file, add a section:

```md
## Log

Implementation issues found during the build (non spec gaps) are tracked in [{spec-basename}-log.md](./{spec-basename}-log.md).
```
