---
title: "Log: match-p3-spec002"
---

# Known Issues

Found via a `frontend-code-review` pass after implementation (already user-confirmed working), not by unit tests. These are issues in the client-layer implementation and its test coverage, not gaps in `match-p3-spec002.md` — logged here for traceability since that spec is what surfaced them. 4 cheap findings were fixed directly in the same pass (team-color fallback warning, unrecognized-archetype warning, magic-number constants, dropped redundant `this.graphics` field) and aren't listed below.

1. **`create()`'s `getMatchState().then()` has no shutdown guard.** If the scene is ever restarted while the fetch is in flight, the stale callback calls `this.add.graphics()`/`this.add.text()` after/during shutdown. Latent today since the scene is entered once from `DevBootScene`; becomes a real risk once a rematch/back-to-lobby flow exists.
   **Status: Deferred.**
2. **`renderUnits`/`renderSoftBlocks`/`renderBombs` allocate a fresh `Graphics` instance per occupant per call.** Not a leak, but would degrade if this path is ever invoked repeatedly (e.g. live polling) instead of once per scene load.
   **Status: Deferred.**
3. **`attachClickLogger`'s inline arrow-function listeners can't be `.off()`'d individually.** Fine now since each listener's lifetime matches its own GameObject, but blocks future per-occupant handler replacement without full recreation.
   **Status: Deferred.**
4. **All occupant hit areas are full 48×48 tiles, and Phaser's `topOnly` input default means only the most-recently-added occupant (bombs, since `renderBombs` runs last) receives clicks when two occupants share a tile.** Implicit ordering dependency, undocumented. Depends on the eventual interaction model — may become moot if tile clicks route through a single tile-level handler instead of per-occupant ones.
   **Status: Resolved (match-p3-spec003).** Confirmed moot on two independent grounds: (a) occupant-vs-occupant — `engine/match.go`'s `IsLandingLegal()` enforces one occupant per tile, so two occupants can never share a tile's hit area; (b) the new allowedTiles overlay introduced by spec003 only ever renders on `OccupantNone` tiles (by construction of `getAllowedTiles()`'s landing-legality filter), so the overlay's hit areas never coexist with an occupant's hit area on the same tile either.
5. **`MatchScene.test.ts` helpers (`gridGraphics`, `occupantGraphics`, `errorText`) use non-null assertions (`mock.results[i]!.value`) guarded only by a comment, not a runtime check.** Test-hygiene nit on scaffolding likely to be reshaped once real interaction tests exist.
   **Status: Deferred.**
6. **No documented convention for when something is "render" vs "draw."** `drawArchetypeIcon` sits alongside the `render*` family with no stated rule; will blur further if more decoration helpers are added.
   **Status: Deferred.**
7. **Duplication across `renderUnits`/`renderSoftBlocks`/`renderBombs`** (tileCenter → add.graphics → fill-shape → attachClickLogger skeleton). Acceptable at 3 call sites per the reviewer's own note; watch-list item if a 4th occupant type is added.
   **Status: Deferred.**
8. **`setup.ts`'s block comment above `createMockGraphics`/`createMockText` mixes "why mockReturnThis" with "how to index mock.results."** The latter logically belongs near the test-file helpers that consume it.
   **Status: Deferred.**
9. **`occupantGraphics(index)` in `MatchScene.test.ts` encodes the units→softBlocks→bombs render-order contract via comment only,** with nothing enforcing it stays in sync with `create()`'s actual call order — silent-drift risk if render order changes for z-ordering reasons.
   **Status: Deferred.**
10. **`Phaser.Geom.Rectangle.Contains = vi.fn()` in `setup.ts` is an unconfigured mock; no test exercises real hit-area containment logic** (tests invoke the captured `pointerdown` handler directly). Reviewer explicitly labeled this a coverage gap, not a defect.
    **Status: Deferred.**
11. **`attachClickLogger`'s `console.log` dumps full Unit/SoftBlock/Bomb objects on every click.** No credential/PII in the current shape; general "don't ship verbose object dumps" note for whenever real interaction UI replaces this scaffolding.
    **Status: Deferred.**
12. **The `'logs roomId and playerTokens on create'` test asserts on literal token values.** Coupling note for whenever the known-non-issue console.log itself gets removed — this test will need updating too.
    **Status: Resolved (match-p3-spec005).** spec005 removes the roomId/playerTokens console.log; implementing it must also update/remove this test's literal-token assertion.
