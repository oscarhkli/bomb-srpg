---
title: "Phase 3.x: Lounge Scene"
---

# Lounge Scene

## Context

Match setup screen, reached from `TitleScene`. Lets the player configure a match before it starts.

## Goal

- Form controls for: stage/map preset, max turn limit, character archetype choices per team.
- `[Start Game]` button validates the form, builds a `GameCfg` payload, calls `POST /match` (via `engine/api.ts`), then transitions to `MatchScene` on success.

_Stub only — flesh out Non-Goal / Scene Entry / Visual Spec / Acceptance Criteria per SPEC_TEMPLATE.md when ready to build._
