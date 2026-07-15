---
title: "Phase 3.7: MatchScene Render-Path Cleanup"
---

# MatchScene Render-Path Cleanup

## Context

Deferred out of `match-p3-spec005.md` so the full game cycle landed first, regardless of performance. spec005's "Game Loop" framing (and its "why `getMatchState()` is called twice" reasoning) has since drifted from the code: `MatchScene` is now event-driven (`beginTurn` / `handleTurnCommand` / `handleResolveTurn`), and the per-turn-start refresh already avoids a wholesale redraw.

Re-grounded against the current code, a small set of redundant, animation-hostile pieces survive after the initial paint — they are the target of this cleanup:

- **One unconditional wholesale redraw:** post-resolve (`refreshFinalSanityCheckAfterResolve`, `MatchScene.ts:582`). It fully destroys and repaints the board *after* `resolveTurnPlayer` has already animated everything to its end-state — a redraw that snaps sprites to rest state (a visible frame-skip once sprites animate).
- **A proactive happy-path diff:** after every *successful* command/resolve, `stateSync.ts` (`turnCommandTargetMatches`, `occupantsMatch`, `extractAppliedTarget`) refetches state and compares it against the server's own just-applied events — i.e. it compares the server to itself.

### Why the proactive diff earns nothing (and what actually does)

The client applies the server's *own* returned events optimistically (tween move, `renderBomb`, `resolveTurnPlayer`); `submitTurnCommand` even persists to server-side `WorkingState` first, so the render is an echo of an authoritative mutation, not a speculative guess. A post-*success* diff can therefore only diverge through a client apply-*code* bug — deterministic, and a test's job, not a runtime one.

The one class of divergence that is *not* a code bug is **network non-determinism** (a lost, duplicated, or reordered response): the server advanced but the client did not observe it, so its `gameState`/graphics lag reality. This is real, and the correct cure is a **reactive** resync — but only on the *error/ambiguous* path, never on the happy path. Two mechanisms already handle it and are **kept**: (1) the per-op `getMatchState()` refetch overwrites `gameState` with truth after each operation; (2) the existing error-path `renderBoard()` resyncs the graphics after a rejected/failed command. This spec keeps the reactive layer and removes only the worthless proactive diff.

## Goal

Establish a single render-path contract (below) and remove the redundant / animation-hostile pieces:

1. **Grid rendered once, in its own persistent layer.** `tileType` is immutable for a match, so the grid is painted at scene entry into a **separate terrain layer** that is never destroyed — distinct from the occupant graphics. No wholesale swap (entry aside), including Reset and error recovery, rebuilds it. *(This terrain layer is also the natural seam for a future mutable-`tileType` feature: a tile change would arrive as a server event and update that one tile in place — the same event-driven pattern as occupants — rather than forcing a full rebuild. Building that is out of scope; see Non-Goal.)*
2. **`renderBoard()` becomes a masked / entry / error-recovery wholesale *occupant* swap** — it rebuilds the occupant layer from truth and leaves the terrain layer untouched; never an unmasked *happy-path* reconciliation tool (see Contract).
3. **Happy-path visuals are in-place.** On a *successful* move (tween), bomb placement (`renderBomb`), or resolve (`resolveTurnPlayer`), the occupant graphics maps mutate in place; no wholesale redraw follows.
4. **Drop the proactive diff; keep the reactive recovery.** Remove `turnCommandTargetMatches`, `extractAppliedTarget`, and the happy-path `renderBoard()` + "out of sync" error. Retain the error-path refetch + `renderBoard()` (network recovery) and the per-op `gameState` refetch.
5. **Relocate `occupantsMatch` to a test oracle.** Its "graphics maps contain exactly the occupants truth says exist" invariant is the assertion for `resolveTurnPlayer` / render-fidelity tests — the right place to catch the apply-code bugs a runtime diff cannot justify guarding. Keep the logic test-side; remove it from the production render path.

## Render-Path Contract

> A full `renderBoard()` (destroy-and-repaint the **occupant** layer from server truth) is a **deliberate wholesale occupant swap**. It runs only at scene entry, behind a screen mask, or on the error/recovery path — never on the *happy* path, because a destroy-and-repaint snaps sprites to rest state (a visible frame-skip mid-animation). The **terrain layer is never in scope** of a swap: it is painted once at entry and persists.

Sanctioned wholesale-redraw callers. **At spec007 implementation time only (a) and (c) are live — (b) is a future caller that lands with spec008** (which this spec is sequenced before), so the implementer should not expect a Reset path to exist yet:

- **(a) `create()` initial paint** — the one time the **terrain layer** is painted, plus the initial occupant paint, at scene entry. **Resume-from-reload** (a browser refresh that wipes in-memory state and re-runs `create()`) is the *same* path: it rehydrates `roomId`/tokens, calls `getMatchState()`, and full-paints whatever truth the server returns. No new render mechanism.
- **(b) Reset (spec008, future)** — its flow dims the canvas, refetches `getMatchState()`, rebuilds occupants, then undims (spec008 §"Reset Button"). The redraw is hidden by the fade, so it is animation-safe. *(Forward-declared here; implemented in spec008. spec007 only defines the contract Reset uses.)*
- **(c) Error-path recovery** — when a command or resolve is **rejected or its response fails**, the client refetches `getMatchState()` and rebuilds occupants to resync (recovers network non-determinism, e.g. a lost/duplicated response). This is an interrupt, not the happy path, so a snap is acceptable.

Between wholesale redraws:

- **The terrain layer persists** (painted once at entry); no operation rebuilds it.
- **The occupant graphics maps (`unitGraphicsById` / `bombGraphicsById` / `softBlockGraphicsById`) are the live on-screen truth**, mutated in place by tweens / `renderBomb` / `resolveTurnPlayer`.
- **`this.gameState` is kept current** via the existing per-op `getMatchState()` refetch (the resolve snapshot and next turn depend on it). This refetch is *retained*; only the proactive diff and the happy-path redraw layered on top of it are removed.

## Non-Goal

- Any gameplay or turn-lifecycle behavior change. In particular, the post-resolve `turnPanel.update()` and `renderResolveButton()` are turn-advance UI, **not** reconciliation — they stay.
- Implementing spec008's Reset itself (this spec only defines the contract Reset will use).
- Session resume / reconnection (persisting `roomId`/tokens, `startTurn()` idempotency on a mid-turn reload). Resume *uses* caller (a) but its lifecycle plumbing is a separate spec — and a turn-lifecycle concern this spec explicitly leaves untouched.
- Sudden-death, victory, and rematch flows are untouched.

---

## Acceptance Criteria

1. Terrain (grid) graphics live in a separate layer, created once per scene entry, and survive **every** wholesale occupant swap (entry aside — Reset, error recovery); no move, bomb, or resolve destroys / recreates them.
2. After a **successful** move or bomb command, the board updates only via the in-place tween / `renderBomb`; no wholesale `renderBoard()` and no "out of sync" message. The `gameState` refetch is retained.
3. After a **successful** `resolveTurn`, the animated end-state left by `resolveTurnPlayer` stands; no wholesale `renderBoard()` follows. `turnPanel` and the resolve button still refresh.
4. When a command or resolve is **rejected or fails** (or playback reports failure), the client surfaces the actual error and resyncs via `getMatchState()` + occupant rebuild. *Design note for spec008:* once Reset exists, error recovery must **not** route through `ResetTurn` — Reset is a user-initiated rollback that would discard the turn's other planned actions.
5. `stateSync.ts` is **deleted** — the proactive helpers (`turnCommandTargetMatches`, `extractAppliedTarget`, and the `AppliedTurnResult` type) leave with it. `occupantsMatch`'s invariant is relocated **test-side** (its own test helper) and survives as the **test oracle** for `resolveTurnPlayer` render-fidelity tests. `make web-test` and `make web-lint` pass.
6. No change to turn lifecycle, sudden-death, victory, or rematch behavior.

## Testing

Harness is Vitest + jsdom with `../engine/api` mocked, so tests assert **behavior** (spy on renderer calls, tween creation, error surfacing), not pixels. Each AC maps to a test:

| AC | Test approach (mocked API, spy-based) |
|---|---|
| 1 — grid once | Capture terrain-layer refs after the initial paint → run a move / bomb / resolve → assert those refs are still alive and the terrain render ran exactly once. (Reset/error swaps are spec008/interrupt paths; assert there too once available that the swap rebuilds occupants but leaves terrain refs alive.) |
| 2 — success is in-place | Mock a successful move → assert `renderBoard`/`drawBoard` **not** called beyond init, a tween **was** created, `showError` **not** called, and the `getMatchState` refetch **did** run |
| 3 — resolve success | Mock a successful resolve → assert no `renderBoard` after playback, but `turnPanel.update` + `renderResolveButton` **were** called |
| 4 — error recovery | Mock `submitTurnCommand` / `resolveTurn` to **reject** → assert `showError(actual error)` and an occupant rebuild **was** triggered (resync). (The "never route through `ResetTurn`" guard is a spec008-time assertion — `resetTurn` isn't wired yet.) |
| 5 — oracle | Removal verified by compile + no lingering imports. `occupantsMatch` is exercised as the **oracle** in a table-driven `resolveTurnPlayer` test: given `(initialState, eventStream)`, play the events, then `assert occupantsMatch(expectedState, maps)` |
| 6 — no lifecycle change | Existing `MatchScene.test.ts` lifecycle tests stay green |

### Limits (do not over-trust green tests)

- **Animation-safety is not unit-testable.** "No visible frame-skip on the happy path" is a visual property; jsdom renders nothing. Confirm it manually via `make web-dev` click-through.
- **Contract drift stays uncovered.** Mocked fixtures are self-authored, so if the Go server's real JSON diverges from `types/api.ts`, every test above stays green while production drifts (the deferred spec001 known issue #4). Closing it needs a real-server contract/integration test, not more unit tests.
