---
title: "Phase 4.8: Fix PlaceBomb Button Problem in Match Scene"
---

# Phase 4.8: Fix PlaceBomb Button Problem in Match Scene

## Context

This spec fix the UI bug found in `MatchScene`. In a new Turn, if a `Unit` has no `Bomb` available, its `placeBombButton` shouldn't be enabled.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

> Note: Unless specified, all vector graphic components are temp representation and will eventually be replaced by pixel art sprites. The dimensions and alignments are just rough estimations - accept they aren't perfect, or even slightly misplaced and out of range.

## Goal

- Disable `placeBombButton` when the `Unit` cannot place a bomb.

## Non-Goal

- N/A

---

## Bomb Button Availability

`placeBombButton` is enabled only when the unit has an unused action this turn **and** holds at least one bomb charge. p3-spec003-match named only the first condition; both are required. Disabled styling is unchanged.

The two conditions expire on different clocks. The per-turn action resets when the turn does; a bomb charge returns only when that bomb detonates. A unit that places a bomb therefore has no charge for the bomb's whole countdown, spanning several of its own turns — during which the button must stay disabled rather than offering a placement the server will reject.

---

## Acceptance Criteria

1. Given a unit places a bomb, when the placement succeeds, then `placeBombButton` is disabled for the remainder of that turn.
2. Given a unit placed a bomb, when that unit's next turn opens and the bomb has not yet detonated, then `placeBombButton` is disabled.
3. Given that bomb has detonated, when the unit's next turn opens, then `placeBombButton` is enabled again.
4. Given a unit that has not acted this turn and holds a bomb charge, when the unit is selected, then `placeBombButton` is enabled.
