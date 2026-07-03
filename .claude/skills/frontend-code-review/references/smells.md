# Naming & code-smell review

This is the lowest-severity category — findings here are almost never "will break in production," they're "will confuse the next reader (including future-you)." Rank accordingly: never let a naming nit outrank a real bug or security issue in the merged report.

## What to check against

- **Backend-mirroring conventions** — read the root `CLAUDE.md`/`AGENTS.md` directly rather than relying on this file's summary of them (they change independently of this skill). Anything in `web/` that names or shapes data in a way that implies a different mental model than the Go backend uses is worth flagging, since that's exactly the kind of mismatch that causes real bugs later even though the finding itself is "just naming."
- **Naming case convention**: TS mirrors Go's identifiers structurally, just camelCase instead of Go's PascalCase (e.g. a Go `UnitID` maps to TS `unitId`, not a renamed or reshaped concept). A TS name that doesn't map cleanly back to its Go counterpart is worth a note.
- **Consistency with existing `web/src` layout.** Scenes live in `web/src/scenes/`, types mirroring the Go API live in `web/src/types/api.ts`, engine/API-client code in `web/src/engine/`. A new file or export that doesn't fit this layout, or a name that collides conceptually with an existing one, is worth a note.
- **Duplication that should be a shared helper.** Two near-identical blocks (e.g. repeated grid↔pixel conversion math, repeated fetch-and-parse boilerplate) that will drift out of sync if only one copy gets fixed later.
- **Dead code / unused exports.** ESLint's `no-unused-vars` catches unused locals already — don't re-report those. Do flag exported functions/types that nothing imports, since the linter won't catch cross-file dead exports.
- **Comments that restate the code** (the project's own style guidance is to avoid comments unless they explain a non-obvious *why*) — if a diff adds a comment that just repeats what the next line does, note it as a light cleanup, not a real finding.
- **Test file drift.** If a diff changes `MatchScene.ts` but not `MatchScene.test.ts` (or vice versa introduces dead test setup), it's worth a mention — this repo pairs scene files with test files and a silent gap is easy to miss.

## Calibration

A naming/smell finding should say *why it will cost someone time later*, not just "this could be named better." If you can't articulate a concrete future confusion or bug it could cause, it's probably not worth including in the report at all — don't pad the findings list with stylistic opinions that have no consequence.
