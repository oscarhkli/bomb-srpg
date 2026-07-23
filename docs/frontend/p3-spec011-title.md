---
title: "Phase 3.11: Title Scene"
---

# Title Scene

## Context

Entry scene when the app boots. Displays the game logo and lets the player choose how to start: enter the match lounge, or (dev-only) jump straight into a mock match without hitting the backend.

## Goal

- Show game title/logo.
- `[Local Match]` button → `LoungeScene`.
- `[Play Offline (Mock)]` button → `MatchScene` with a client-only mock `GameState`, no API calls (useful for iterating on rendering without a running backend).

_Stub only — flesh out Non-Goal / Scene Entry / Visual Spec / Acceptance Criteria per SPEC_TEMPLATE.md when ready to build._
