---
name: frontend-code-review
description: Review frontend code changes under web/ (Phaser 4 + TypeScript) for bugs, client-side security issues, and naming/code-smell cleanups, with an optional spec-conformance pass. Use whenever the user asks to review, audit, or check frontend/web/Phaser/TypeScript code changes. Do NOT use for backend/engine/server (Go) review — that's plain /code-review's territory; this skill only looks at paths under web/.
arguments: [specPath]
argument-hint: "[specPath]"
---

# Frontend Code Review

Domain-specific code review for the `web/` directory of this repo (Phaser 4.2.0 + TypeScript, Vite/Vitest). The user is a backend engineer (Go) with **no frontend background**, so findings must be explained in plain language, not just flagged.

This skill reuses the review *pattern* of `/code-review` (diff-scoped, severity-ranked findings via `ReportFindings`) but adds things generic review doesn't have: Phaser/TS domain knowledge, a hard scope filter to `web/`, a fixed category split so nothing gets skipped just because it isn't a "bug," and an optional pass that checks the diff against its `docs/frontend` spec.

**This skill never auto-fixes and never posts PR/GitHub comments.** Its job is to produce a discussion list the user reads and decides on — not to touch code or GitHub.

## Step 1 — Resolve the review target

Figure out what to diff, in this order:

1. If the user names a PR number, fetch it: `gh pr diff <n> -- web/`.
2. Else if `git diff --cached -- web/` is non-empty, review **staged** changes.
3. Else if `git diff -- web/` is non-empty, review **unstaged** changes.
4. Else, review the current branch against its base: `git diff main...HEAD -- web/`.

Always scope to `web/` only — pass `-- web/` (or filter the PR diff to `web/` paths) so Go engine/server changes never enter this review. If the resolved diff has **no files under `web/`**, tell the user there's nothing to review here and suggest `/code-review` for the rest of the diff. Stop.

If the target is genuinely ambiguous (e.g. both staged and unstaged changes exist and it's unclear which the user means), ask — don't guess silently.

## Step 2 — Decide whether a spec-conformance pass applies

1. If `$specPath` was given, use it directly.
2. Otherwise, read `docs/frontend/toc.yaml` and match its `category` field against the diff's changed file names (e.g. `web/src/scenes/MatchScene.ts` → category `MatchScene`), considering only entries with `status: Ready` — that's the current, actively-being-built contract. (`Done` means shipped/merged history, not something in-flight work is being checked against; `Draft`/`Parked Draft` aren't authoritative yet. Revisit this if more statuses get added later.)
3. If exactly one `Ready` spec matches, use it. If more than one plausibly matches, **ask the user which one** — don't guess. If none match, skip this pass and say so in the final wrap-up rather than silently dropping it.

`references/spec-conformance.md` covers what to check once a spec is in hand — read it now if Step 2 found one.

## Step 3 — Read the diff and load domain references

Read the full diff. Then, based on what's actually touched, skim the relevant reference file(s) below — don't load references for categories the diff can't possibly trigger:

- `references/phaser-idioms.md` — review lens for Phaser lifecycle/event/timer issues; points to the installed per-topic Phaser 4 skills (`~/.claude/skills/scenes`, `events-system`, `tweens`, etc.) for authoritative API behavior.
- `references/typescript-strict.md` — `any`/unsafe-cast patterns, non-null assertions, structural typing footguns.
- `references/security.md` — client-side XSS/DOM-injection surface, unsafe URL/fetch construction, secret/token handling in browser code.
- `references/smells.md` — naming/duplication conventions specific to `web/`'s own layout (not a restatement of `CLAUDE.md`/`AGENTS.md` — read those directly for backend-mirroring conventions like state sandboxing or ID encoding).
- `references/spec-conformance.md` — only relevant if Step 2 found a spec.
- `references/known-non-issues.md` — if present, a running list of patterns the user has already told this skill are false alarms. Check the diff against it before flagging anything similar. See Step 6.

## Step 4 — Parallel passes

Spawn `general-purpose` subagents in parallel (single message, multiple `Agent` calls) — three always, plus a fourth if Step 2 found a spec to check against. Each gets: the diff text (or the list of changed files + how to get the diff themselves), a pointer to read its one reference file plus `known-non-issues.md` if it exists, and this instruction shape:

- **Correctness & Phaser/TS idioms** — reads `phaser-idioms.md` + `typescript-strict.md`, consulting the linked Phaser topic skills as needed. Looks for actual bugs (state desync, event-listener leaks causing memory growth across scene transitions, race conditions in async Phaser callbacks) and idiom violations that will bite later (raw `any`, unsafe `as` casts, ignoring Phaser 4's destroy/cleanup contract).
- **Security** — reads `security.md`. Looks for DOM-injection surface (`innerHTML`, `document.write`, `eval`/`Function`), unsafe `fetch`/URL construction, and secrets/tokens landing in client-visible code, `localStorage`, or logs.
- **Naming & smells** — reads `smells.md`. Looks for naming inconsistencies, duplicated logic that should share a helper, dead code, and drift from this repo's conventions.
- **Spec conformance** (only if Step 2 found a spec) — reads `spec-conformance.md` plus the spec file itself (and its `-log.md` sibling if present — that file is maintained by the `frontend-sdd` skill during implementation, not something this skill creates). Looks for Non-Goal creep, silent contradictions of the spec, and unacknowledged gaps against `## Limitation`.

Each subagent should report back a plain list of `{file, line, category, one-sentence defect, why it matters}` — instruct them explicitly **not** to run `ReportFindings` themselves (only the top-level orchestrator does that once, after merging).

Wait for notifications as they arrive; don't manufacture manual sleep/poll loops. If one pass clearly isn't coming back once the others have returned, don't keep waiting — do that category's pass yourself inline using its reference file, and say in the final report that it was done inline because the subagent didn't return.

## Step 5 — Merge and report

Collect all subagents' findings. Deduplicate anything two categories both caught (keep the more specific write-up). Rank most-severe first: actual bugs, security issues, and Non-Goal/spec-contradiction findings outrank naming/style and partial-coverage notes.

If two findings from different categories conflict (e.g. security flags something the spec explicitly requires), don't pick a winner — report both, note the conflict explicitly, and let the user decide.

Present the merged, ranked list as plain CLI text — don't call `ReportFindings`. The user works entirely in the terminal and prefers to skip the extra tool call to save tokens, so plain text is the default output here, not just a fallback for when the tool happens to be unavailable.

After the tool call, add a short plain-language wrap-up in your own words (a sentence or two per finding at most) — the user doesn't know Phaser or TS well enough to parse a terse bug summary unaided. Ask which findings they want to dig into or act on; don't start editing files unless they ask.

## Step 6 — Learn from false positives

If the user tells you a finding isn't actually a problem, ask whether to record it so future reviews stop re-flagging the same pattern. If they say yes, append an entry to `references/known-non-issues.md` (create it if it doesn't exist yet) capturing: the pattern, the file/context it applies to (or "applies generally" if it's not context-specific), and *why* it's not actually an issue in this codebase. Keep entries short — this file is meant to stay skimmable, not become a second findings log.

## Notes

- If `gh` isn't authenticated or the PR fetch fails, say so and fall back to asking for a local diff instead of guessing.
- If the diff is large, it's fine for each subagent to re-derive its own diff via `git`/`gh` rather than you pasting the whole thing into the prompt — pass the exact command instead of the raw text when it's long.
- Severity intuition: a security or state-desync bug that could ship is high; a Phaser leak that degrades over a long session is medium; naming/smell items are low unless they actively cause confusion with this codebase's established conventions.
