---
title: "Phase 4.10: Refine 2-phase GameEvents Handling in ResolveTurn"
---

# Phase 4.10: Refine 2-phase GameEvents Handling in ResolveTurn

## Context

`Match.ResolveTurn()` produces two phases of events: the planning events submitted through `ApplyTurnCommand`, followed by the resolution events produced by bomb detonation. The engine returns them as two slices, but `ServerStateManager.ResolveTurn` concatenates them, so `/resolve` hands the client one flat array with no marked seam.

`MatchScene` copes by classifying on `GameEvtType`. `resolveTurnPlayer` handles only `bombCountdownUpdated`, `bombExploded`, `unitDamaged`, `unitDied` and `softBlockDestroyed`, so the Player's own `unitMoved`/`bombPlaced` prefix falls through the switch and is dropped correctly, because those were already animated when the command was submitted.

That classification holds only while each `GameEvtType` belongs to exactly one phase. It is true today, but will be invalid when Skills are introduced in future.

[p4-spec009-cpu](p4-spec009-cpu.md) already consumes the two slices on the VS CPU path via `/cpu-status/consume`. This spec brings the VS Human path to the same shape.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- `MatchScene` drops the already-animated planning events by position, not by event type.
- `resolveTurnPlayer` no longer infers phase from `GameEvtType`.
- VS Human and VS CPU consume the same response shape.

## Non-Goal

- Per-event phase stamping, or an append-only event journal.
- Any change to how an individual event is animated.

## Scene Entry

No change.

---

## Prerequisites

Landed outside this spec:

- `ServerStateManager.ResolveTurn` stops concatenating and returns both slices.
- `HandleResolveTurn` emits them as two fields.
- `web/src/types/api.ts` replaces `resolveTurn()`'s bare `GameEvent[]` with `ResolveTurnResponse`

## Match Scene

`MatchScene`'s resolve handler passes **only** `resolveTurnGameEvents` to `playResolveTurnEvents`. `planGameEvents` is skiped. This is what `submitTurnCommand` already animated during planning.

The discard is deliberate and positional. It does not depend on which types appear in either array, which is the property the current filter lacks.

Error handling, the generation guard, and the `turnPanel` refresh are unchanged.

## Resolve Turn Player

`resolveTurnPlayer` keeps its per-type cases but stops relying on fall-through as classification:

- Types it deliberately ignores are named in an explicit ignore-list, so the assumption is stated in code rather than implied by omission.
- Any type outside both the handled set and the ignore-list logs a dev-mode warning naming the type. A future phase-spanning event then surfaces instead of misrendering.

## VS CPU

Unchanged. The CPU branch introduced in Phase 4.9 already consumes two arrays; after this spec both paths share one shape and the plan/resolve split is expressed identically on each.

---

## Acceptance Criteria

1. Given a Player turn with a move and a bomb placement, when the turn is resolved, then each unit animates its move exactly once.
2. Given a Player turn placing a bomb that detonates in the same turn, when the turn is resolved, then the explosion animates and no planning-phase `unitMoved` is replayed.
3. Given a turn whose resolution produces no events, when the turn is resolved, then `resolveTurnGameEvents` is an empty array and the Player advances to the next turn without error.
4. Given a VS CPU match, when the CPU turn is consumed, then rendering is unchanged from p4-spec008.
5. Given `resolveTurnPlayer` receives an event type in neither the handled set nor the ignore-list, when running in dev mode, then a warning is logged naming the type.
