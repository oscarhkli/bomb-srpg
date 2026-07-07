---
title: "Phase 3.x: Turn Lifecycle Wiring (startTurn)"
---

# Turn Lifecycle Wiring (startTurn)

## Context

`MatchScene` never calls `startTurn()` (`engine/api.ts`) today. Per `AGENTS.md`'s Turn Lifecycle rules, the client must explicitly call `POST /match/start-turn` for every turn, including Turn 1, to evaluate sudden-death / hazard injection — this has been a silent gap since spec001/002. `match-p3-spec003.md` (Move/PlaceBomb) works around it by re-deriving `initToken()` once per `TurnCommandPanel` open instead of once per turn boundary.

## Goal

- Wire `startTurn()` into the turn flow at the correct point(s).
- Move `initToken(playerTokens[activeTeam - 1])` to fire once per `startTurn()` resolution instead of once per panel-open, since `activeTeam` is stable between turns.

_Stub only — flesh out Non-Goal / Scene Entry / Visual Spec / Acceptance Criteria per SPEC_TEMPLATE.md when ready to build._
