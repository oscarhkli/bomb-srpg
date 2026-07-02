---
title: "Phase 3.5: Render Bombs + Camera Navigation"
---

# Render Units and Bombs + Camera Navigation

## Context

Phase 3.1 renders the static `grid` tiles. This spec adds the dynamic layer: `units` and `bombs` drawn on top of the grid, plus pan and zoom so the player can navigate large grids (up to 15×15).

## Goal

- `MatchScene` renders each `Unit` and `Bomb` from `GameState` as procedural shapes on the correct `Tile`.
- Player can pan the camera by click-dragging and zoom with scroll wheel.

## Non-Goal

- Unit selection or command input (see spec003).
- Animations or tweens.
- HUD / status panel.
- Rendering of `activeTeam`, `softBlocks`, or `turnCommands`.

## Scene Entry

_TODO: fill in_

---

## Layout

| Interaction | Behaviour |
|---|---|
| Click + drag | Pan the world camera |
| Scroll wheel | Zoom in / out (range: 0.5× – 2×) |

Camera pan is bounded to the grid extents so the player cannot drag the grid fully off-screen.

## Data Fetching

No change from spec001 — `getMatchState()` is called once on `create()`. Units and bombs are drawn from the same `GameState` snapshot.

## Visual Spec

### Unit

Each `Unit` is drawn as a **32×32px** shape centered on its `Tile`, using `team` as the color index.

| `team` | Fill color |
|---|---|
| 0 | Blue |
| 1 | Red |

Archetype shape is drawn inside the fill (white stroke):

| Archetype (by `type` string) | Shape |
|---|---|
| King | Circle |
| Soldier | Triangle |
| (fallback) | Square |

### Bomb

Each `Bomb` is drawn as a **16×16px** dark circle centered on its `Tile`.
`countdown` is rendered as white text above the circle.

---

## Acceptance Criteria

1. Given a `GameState` with two teams, each `Unit` renders on the correct tile with the correct team color and archetype shape.
2. Each `Bomb` renders on the correct tile with a visible countdown number.
3. On a 15×15 grid, the player can pan and zoom to reach any corner.
