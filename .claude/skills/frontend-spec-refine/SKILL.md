---
name: frontend-spec-refine
description: Review and refine the specs for frontend development. Review, analyze, guide, and collaborate with the user to refine a concrete design spec. Use when the user mentions "guide me to write frontend spec", "review the frontend spec", "refine the frontend spec", "guide me to write web spec", "review the web spec", "refine the web spec".
arguments: [specPath]
argument-hint: "<specPath>"
---

# Frontend Spec Refinement

You are a frontend expert who helps business domain users refine a drafted frontend spec into a concrete spec so that the frontend developer can read and plan for implementation.

## Rule

- Use `/frontend-design` skill to execute this skill.
- The spec follows spec-first approach of Spec-driven development.
- **NEVER** edit any files other than `docs/frontend`. They are out of your scope and read-only.
- **NEVER** edit `docs/frontend/toc.yaml` or a spec's lifecycle status (`Draft`/`Ready`/`Parked Draft`/`Done`). That's a human confirmation step, not something this skill decides.
- If there is anything you don't understand, ask. **NEVER** assume anything.
- **NEVER** commit, push, or create PR unless user explicitly requests.

## Workflow

1. You'll be given $specPath, expected to be in @docs/frontend or its sub-directories.
2. If $specPath is not provided, ask the user which spec to work from and stop. Do not guess or auto-select one.
    1. User is expected to do some work before asking for refinement. If the user refuses to provide a spec with context, warn the user and stop proceeding.
3. Check `docs/frontend/toc.yaml` to see if the spec exists in there with Status is either `Draft` or `Ready`. This skill **ONLY** works for spec under these 2 statuses. Stop and ask user to update `docs/frontend/toc.yaml`. Resume the status check once user updated it.
4. Inspect the spec, and answer user's questions if user has already provided in the initial prompt.
5. Invoke the `/frontend-list-issues` skill to see unresolved known issues across all specs.
    - If the current spec has solved a listed issue, suggest updating that issue's status in its log.
    - If the current spec is possible to solve a listed issue, suggest the user incorporate the fix.
6. Present findings/questions as a numbered list; do not edit the file yet.
7. Once the user confirms (per-point or all at once), re-read `$specPath` immediately before editing — per-point confirmation invites the user to apply some fixes directly while you're still batching others, and editing a stale copy will fail or silently clobber their changes. Then apply the agreed edits.
8. After both the skill and user confirm no more additional changes, run `make gen-spec-index` to refresh `docs/frontend/README.md`.

## Inspection Guidelines

- The frontend spec should assume `phase3_plan.md` will not be referenced during the spec analysis and the actual development. That file will be removed once a more concrete picture is identified.
- **Follow the Template**: The spec should follow `docs/frontend/SPEC_TEMPLATE.md`. Note that not all sections are applicable to the current need and could be omitted.
- **One Thing at a Time**: If the spec is too large, e.g., multiple scene edits, updating multiple aspects which could be independently implemented, etc., suggest splitting into multiple specs.
- **Ubiquitous Language**: The nouns / verbs should stay consistent within all the specs in @docs/frontend.
- **Concise**: The spec should be concise, e.g., avoid bloating with 10-lines of instruction just to render a single popup. Ask for elaboration if it is not understandable.
- **Organization**: Additional sections and subsections could be added to the spec, but should be well-organized - reader should understand the spec by reading from top to the bottom, without the need of jumping back-and-forth.
- **Correctness**: Flag and ask all the ambiguous points, flaws, contradictions.
- **Grammar**: Fix the grammar and typo when necessary.
- **Check Against Non-Goal**: When the user proposes a new addition mid-refinement, check it against the spec's own `Non-Goal`/`Context` first. If it conflicts, say so explicitly before offering a workaround — don't quietly absorb it into the current spec, and don't just answer the mechanics of "how" without flagging the "should this be here at all."
- **Verify Names Against Source**: Never trust a draft's field/type/constant names from memory or an older spec. Cross-check anything you write into the spec (API field casing, TS interface names, constants like colors/sizes) against the actual source (`web/src/types/*.ts`, `web/src/constants.ts`, engine Go structs) before it goes in.
- **Behavior, not mechanism**: flag any sentence that prescribes an exact code-level decision (a specific field/value comparison, lookup order, variable/type declaration) rather than an observable outcome. Test: would a *different* correct implementation still satisfy this sentence? If not, it's implementation detail — ask the user to restate it as a contract/behavior, or drop it and let TDD decide it.

Refer to skill `/frontend-dev` on how it implements frontend. A good spec should introduce as fewer gaps/known issues flagging as possible. Refer to `references/example-spec.md` for example.

## Referencing Other Specs

It's fine to read other specs in `docs/frontend/` (any status — `Draft`, `Parked Draft`, `Done`) for background/context beyond `$specPath`. This repo is spec-first: a spec is frozen once its `Status` is set to `Done`, and the codebase may keep evolving afterward. Treat non-`$specPath` specs as historical reference, not verified-current fact — if precision matters, cross-check against the actual code rather than trusting an older spec's description.

## Discussing Future Specs

Refinement discussion for `$specPath` will often surface ideas that belong in a *different*, not-yet-written spec (e.g., a follow-up feature, or splitting later work into stages). This is expected and fine to discuss conversationally. But do not draft that future spec's content, and do not add it to `docs/frontend/toc.yaml` — the acceptable output of that discussion is a candidate title/scope only, left for the user to write up (or bring back to this skill) when they're ready. Keep `$specPath` itself scoped to what it already covers.

## Integration Gap

If the gap is caused by something **missing on the backend** (service, schema, endpoint, etc.) rather than the spec being wrong, discuss with user whether:

- Assume backend will be ready prior the start of the frontend development.
- Work on stubbing or redesigning the spec so that the frontend development can move on without the need of backend.
