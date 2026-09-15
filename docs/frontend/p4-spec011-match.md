---
title: "Phase 4.11: Fix Various Rendering Problem in VS-CPU MatchScene"
---

# Phase 4.11: Fix Various Rendering Problem in VS-CPU MatchScene

## Context

Manual testing of the VS CPU flow surfaced two problems in `runCpuTurn()` (`web/src/scenes/MatchScene.ts`), both isolated to the CPU branch — the VS Human branch (`handleTurnCommand` / `handleResolveTurn`) is unaffected.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- CPU turns never surface a "references unknown bombId" validation error.
- CPU `planGameEvents` animate one at a time, in order, matching how a human's own move-then-bomb reads.
- CPU `resolveTurnGameEvents` playback is driven by the same up-to-date state a Human turn's `/resolve` already gets, so both paths hit the same code with the same guarantees.

## Non-Goal

- Changing `resolveTurnPlayer`'s per-type animation logic itself.
- Any change to the VS Human path — it already resyncs `this.gameState` after every submitted command, so it doesn't have either problem below.

## Scene Entry

No change.

---

## Issue 1: `resolveTurnGameEvents` validated against a stale snapshot

**Symptom:** after a CPU turn that moves and places a bomb, the client shows `bombCountdownUpdated event references unknown bombId <id>`, even though the bomb is genuinely on the board.

**Root cause:** `runCpuTurn()` plays `planGameEvents` through `applyGameEvent()`. For `bombPlaced`, `applyBombPlaced()` only renders the bomb sprite (`renderBomb()`) — it never adds the bomb to `this.gameState.bombs`. Right after, `runCpuTurn()` calls `playResolveTurnEvents(resolveTurnGameEvents, { gameStateSnapshot: this.gameState, ... })`, so `resolveTurnPlayer`'s `validate()` checks the incoming `bombCountdownUpdated` event's `bombId` against a `this.gameState` that predates the CPU's own bomb placement.

The VS Human path doesn't have this gap: `handleTurnCommand()` calls `resyncFromServer()` after every submitted command, so by the time `/resolve` plays back, `this.gameState` already reflects everything planned. `runCpuTurn()` has no equivalent resync between the plan phase and the resolve phase.

**Fix direction:** by the time `ConsumeCPUStatus` returns `TurnPhaseReady`, the server has already run the full turn — planning and resolution — inside one locked call (`runCPUTurn`, `server/server_manager.go`). So a `getMatchState()` fetched at that point already reflects the final, post-resolve truth. `runCpuTurn()` should resync `this.gameState` from the server after the `planGameEvents` playback (replacing the visual-only `applyGameEvent()` loop's job of keeping state in sync) and before constructing the `gameStateSnapshot` passed to `playResolveTurnEvents`. This both fixes the validation gap and is what makes Issue 3 below true by construction — CPU and Human end up feeding `resolveTurnPlayer` a `gameStateSnapshot` under the same contract (current server truth, not a pre-turn cache).

## Issue 2: `planGameEvents` animate concurrently instead of sequentially

**Symptom:** for a CPU turn that moves then places a bomb, the unit move tween and the bomb-placement render start at the same time instead of one after the other.

**Root cause:** the `for (const event of planGameEvents) { this.applyGameEvent(event); }` loop in `runCpuTurn()` calls `applyGameEvent()` synchronously for every event with no `await` between iterations. `applyUnitMoved()` starts a tween and returns immediately, so by the time the next event (`bombPlaced`) is applied, the move tween is still animating.

**Fix direction:** step through `planGameEvents` one at a time, holding for a fixed delay between each so the unit finishes moving before the next event starts rendering. Add a new constant next to `CPU_PLAN_RESOLVE_HOLD_MS` in `web/src/constants.ts` (e.g. `CPU_PLAN_EVENT_STEP_MS`) for the per-event delay, defaulting to 500ms.

## Issue 3: CPU's `resolveTurnGameEvents` playback should match Human's

Already covered by the Issue 1 fix: `playResolveTurnEvents` is shared code (see [p4-spec010-match](p4-spec010-match.md)) — once `runCpuTurn()` hands it a `gameStateSnapshot` resynced from the server the same way the Human path's is kept current, both paths animate `resolveTurnGameEvents` identically. No separate change is needed in `resolveTurnPlayer` itself.

---

## Acceptance Criteria

1. Given a CPU turn that moves a unit and places a bomb that survives resolution, when the turn is consumed, then no "unknown bombId" (or any other) validation error is shown.
2. Given a CPU turn with N `planGameEvents`, when they animate, then each event's rendering starts only after the previous event's `CPU_PLAN_EVENT_STEP_MS` hold has elapsed — no two plan events animate concurrently.
3. Given a CPU turn, when `resolveTurnGameEvents` plays back, then `playResolveTurnEvents` receives a `gameStateSnapshot` reflecting the server's post-resolve state, matching the guarantee the Human path already has.
