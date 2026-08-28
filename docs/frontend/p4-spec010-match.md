---
title: "Phase 4.10: Refine 2-phase GameEvents Handling in ResolveTurn"
---

# Phase 4.10: Refine 2-phase GameEvents Handling in ResolveTurn

## Context

`Match.ResolveTurn()` produces two phases of events: the planning events submitted through `ApplyTurnCommand`, followed by the resolution events produced by bomb detonation. The engine returns them as two slices, but `ServerStateManager.ResolveTurn` concatenates them, so `/resolve` hands the client one flat array with no marked seam.

`MatchScene` copes by classifying on `GameEvtType`. `resolveTurnPlayer` handles only `bombCountdownUpdated`, `bombExploded`, `unitDamaged`, `unitDied` and `softBlockDestroyed`, so the Player's own `unitMoved`/`bombPlaced` prefix falls through the switch and is dropped — correctly, because those were already animated when the command was submitted.

That classification holds only while each `GameEvtType` belongs to exactly one phase. It is true today. This spec is **not scheduled work**; see Trigger.

[p4-spec009-cpu](p4-spec009-cpu.md) already consumes the two slices on the VS CPU path via `/cpu-status/consume`. This spec brings the VS Human path to the same shape.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Trigger

Implement this spec when the first engine change makes an event type span both phases. Two known shapes:

- A skill that alters a bomb's countdown during planning. `bombCountdownUpdated` then reaches `resolveTurnPlayer` and animates as a resolution event, on top of already being animated at command-submission time.
- A forced move during resolution — knockback, or a unit displaced by a blast. `unitMoved` is then silently dropped, and the unit teleports on the next state sync.

Neither exists yet: every archetype in `engine/presets.go` is `PresetSkills: SkillNone`. Until one lands, the type filter is correct and this spec stays Draft.

## Goal

- `MatchScene` drops the already-animated planning events by position, not by event type.
- `resolveTurnPlayer` no longer infers phase from `GameEvtType`.
- VS Human and VS CPU consume the same response shape.

## Non-Goal

- Per-event phase stamping, or an append-only event journal. Both remain deferred.
- Any change to how an individual event is animated. This spec changes only which events reach the animator.

## Scene Entry

No change.

---

## Prerequisites

Landed outside this spec:

- `ServerStateManager.ResolveTurn` stops concatenating and returns both slices.
- `HandleResolveTurn` emits them as two fields. Both are non-nil at the JSON boundary, guaranteed by the function that produces them, per `AGENTS.md`.
- `web/src/types/api.ts` replaces `resolveTurn()`'s bare `GameEvent[]` with:

```ts
export interface ResolveTurnResponse {
  planGameEvents: GameEvent[];
  resolveTurnGameEvents: GameEvent[];
}
```

## Match Scene

`MatchScene`'s resolve handler passes **only** `resolveTurnGameEvents` to `playResolveTurnEvents`. `planGameEvents` is discarded — it is the echo of what `submitTurnCommand` already animated during planning.

The discard is deliberate and positional. It does not depend on which types appear in either array, which is the property the current filter lacks.

Error handling, the generation guard, and the `turnPanel` refresh are unchanged.

## Resolve Turn Player

`resolveTurnPlayer` keeps its per-type cases but stops relying on fall-through as classification:

- Types it deliberately ignores are named in an explicit ignore-list, so the assumption is stated in code rather than implied by omission.
- Any type outside both the handled set and the ignore-list logs a dev-mode warning naming the type. A future phase-spanning event then surfaces instead of misrendering.

## VS CPU

Unchanged. The CPU branch introduced in p4-spec008 already consumes two arrays; after this spec both paths share one shape and the plan/resolve split is expressed identically on each.

---

## Acceptance Criteria

1. Given a Player turn with a move and a bomb placement, when the turn is resolved, then each unit animates its move exactly once.
2. Given a Player turn placing a bomb that detonates in the same turn, when the turn is resolved, then the explosion animates and no planning-phase `unitMoved` is replayed.
3. Given a turn whose resolution produces no events, when the turn is resolved, then `resolveTurnGameEvents` is an empty array and the Player advances to the next turn without error.
4. Given a VS CPU match, when the CPU turn is consumed, then rendering is unchanged from p4-spec008.
5. Given `resolveTurnPlayer` receives an event type in neither the handled set nor the ignore-list, when running in dev mode, then a warning is logged naming the type.
