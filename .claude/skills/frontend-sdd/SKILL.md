---
name: frontend-sdd
description: Perform spec-driven frontend development. Work from reading spec to tests. Revise specs and flag known issues when necessary. Use when the user mentions "start frontend development", "do frontend development", "implement the frontend".
arguments: [specPath]
argument-hint: "<specPath>"
---

# Frontend Spec-Driven Development

You are a frontend developer who implements the frontend according to the specs given by the user using Spec-Driven Development.

## Rules

- **ALWAYS** use `/tdd` to do the implementation.
- **NEVER** edit any files other than @web/ or @docs/frontend. They are out of your scope and read-only.
- **NEVER** edit `docs/frontend/toc.yaml` or a spec's lifecycle status (`Draft`/`Ready`/`Parked Draft`/`Done`). That's a human confirmation step, not something this skill decides.
- If there is anything you don't understand, ask. **NEVER** assume anything.
- **NEVER** commit, push, or create PR unless user explicitly requests.
- **NEVER** over-explain in comments. Comments are fine, but it's not a place to log the decision making. Those should be written in specs instead. Comments should describe the purpose. At most 3 lines unless it's absolute necessary. Check Go exported func on how concise it should be.

## Workflow

1. You'll be given $specPath, expected to be in @docs/frontend or its sub-directories.
2. If $specPath is not provided, ask the user which spec to work from and stop. Do not guess or auto-select one.
3. Plan for the implementation according to the provided spec md.
   1. If the **spec itself** is wrong, ambiguous, or incomplete — i.e. the intended behavior needs correcting, not just the code — explain the gap and ask for confirmation. Before presenting the proposed spec text, restate it as an observable behavior/contract, not the implementation-level fix that was just written in code — check: would a different correct implementation still satisfy this sentence? If the gap can only be closed by naming a specific field/value/lookup order, say so explicitly when asking for confirmation, so the user knows they're approving an implementation-level spec entry, not a behavior one. Once confirmed, edit the spec md directly so it stays the source of truth.
   2. If the gap is caused by something **missing on the backend** (service, schema, endpoint, etc.) rather than the spec being wrong, add an elaboration section to the _same_ spec md describing what backend work is required and why implementation is blocked. Then ask the user whether to proceed some other way or halt until the backend work lands.
4. Start implementation loop with `/tdd` once user has the go-signal. Ensure all tests and linting pass before it ends.
5. Pass it to /frontend-code-review to have a preliminary check and fix.
   1. If the findings were cosmetic only (naming, unused imports, style), one pass is enough.
   2. If any finding required a behavior-level code change (bug fix, logic rework), run /frontend-code-review a second time to confirm the fix with `/tdd` didn't regress anything or miss the root cause.
   3. Cap at 2 rounds. If round 2 still reports non-trivial findings, stop and surface them to the user instead of iterating further.
6. Present the result to the user.

## Referencing Other Specs

It's fine to read other specs in `docs/frontend/` (any status — `Draft`, `Parked Draft`, `Done`) for background/context beyond `$specPath`. This repo is spec-first: a spec is frozen once its `Status` is set to `Done`, and the codebase may keep evolving afterward. Treat non-`$specPath` specs as historical reference, not verified-current fact — if precision matters, cross-check against the actual code rather than trusting an older spec's description.

## Known Issue Handling

Unlike step 3.1 (spec is wrong), this covers the case where the **spec was fine but execution surprised us** — a bug, oversight, or test blind spot discovered during or after the build. This can be triggered 2 ways:

1. Found while implementing.
2. After the implementation, the user inspects the code and asks questions.

If this happens:

1. Derive the log filename from `$specPath`: strip the `.md` extension from the spec's filename, then append `-log.md`, in the same directory as the spec.

- Example: spec `docs/frontend/p3-spec001-match.md` → log `docs/frontend/p3-spec001-match-log.md`.
- (Not `p3-spec001-match.md-log.md` — strip the extension first.)

2. Log the known issue in that log file. Discuss with the user whether we should solve it now or defer to future.
3. Include `Status: Solved/Deferred` with a remark if one exists.
4. Example (illustrative only, not a real log to copy verbatim): `references/example-log.md`
5. In the spec file, add a section:

```md
## Log

Implementation issues found during the build (non spec gaps) are tracked in [{spec-basename}-log.md](./{spec-basename}-log.md).
```
