# Spec conformance review

This repo is spec-first (see `docs/frontend/`, and the `frontend-sdd`/`frontend-spec-refine` skills). A spec's `## Goal` / `## Non-Goal` / `## Limitation` sections are the contract for what a given piece of frontend work is supposed to (and isn't supposed to) do. This pass checks the diff against that contract — it's a different kind of finding than a bug or a smell: **"this doesn't match what was specified,"** not "this is broken."

Spec discovery (which file to check against) already happened in SKILL.md Step 2 — this file only covers what to do once you have one.

Always read the spec file as it currently exists on disk, including any uncommitted edits within this same diff — that's the "latest" contract, not the remote/base-branch version. If the diff edits the spec itself, treat that as a deliberate co-edit, not staleness.

## What to check

- **Goal coverage**: does the diff actually implement what `## Goal` describes? Partial implementation isn't automatically wrong (specs get built incrementally) but is worth surfacing so the user knows what's still open.
- **Non-Goal creep**: does the diff implement something explicitly listed under `## Non-Goal`? This is a real finding — scope creep beyond what was specified, which either means the spec is stale and should be revised, or the implementation went further than intended.
- **Limitation acknowledgment**: if `## Limitation` documents a known-blocked capability, check the diff isn't quietly attempting that blocked capability in a half-working way.
- **Known issues / logs**: check for a sibling `{spec-basename}-log.md` (e.g. `match-p3-spec001-log.md` next to `match-p3-spec001.md`) — `frontend-sdd` creates and maintains this file while implementing against the spec. If an issue is already logged there with `Status: Deferred`, don't re-report it as a fresh finding — reference the existing log entry instead so the user isn't asked to re-litigate a decision already made.
- **Spec staleness**: only applies if the diff does *not* already edit the spec. If an unedited spec contradicts the code's current behavior, flag that it may need updating — a case for `frontend-spec-refine`, not something to silently paper over.

## Severity calibration

- Non-Goal creep or a direct contradiction of the spec's stated behavior: high — this is a correctness-of-intent bug, not a style nit.
- Partial Goal coverage on a spec that's still being actively built: usually informational, not a "finding" at all — mention it in the wrap-up, not as a ranked defect, unless the gap is silent (nothing in the diff or its tests acknowledges the gap the way `## Limitation` sections do).
- Spec staleness: medium — worth fixing, but it's a documentation task, not a runtime risk.
