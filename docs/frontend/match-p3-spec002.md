---
title: "Phase 3.2: Render Units, SoftBlocks and Bombs"
---

# Render Units, SoftBlocks and Bombs

## Context

Phase 3.1 renders the static `grid` tiles. This spec adds the dynamic layer for 3 of the Occupants: `units`, `softBlocks` and `bombs` drawn on top of the grid.

## Goal

- Log `roomId`, `playerTokens` once it's obtained though browser DevTool.
- `MatchScene` renders each `Unit`, `SoftBlock` and `Bombs` from `GameState` as procedural shapes on the correct `Tile`.

## Non-Goal

- Unit selection or command input.
- Animations or tweens.
- Polished unit sprites.
- HUD / status panel.
- Rendering of `activeTeam`, or `turnCommands`.

## Limitation

- As currently many components are missing, many testing can't be easily done at the moment. They'll be revisited later when we work on resolveTurn.
  - Bomb countdown and detonation
  - Unit died
  - SoftBlock destroyed
  - PlaceBomb

## Scene Entry

No change from spec001.

## Data Fetching

No change from spec001 — `getMatchState()` is called once on `create()`. Units and SoftBlock are drawn from the same `GameState` snapshot.

Once `roomId` and `playerTokens` are obtained, log them though `console.log()`. This is useful in early stage debugging.

## Visual Spec of Debug Panel

### Location of Occupants

Refer to `references/state.json` for the sample `getMatchState()` json.

For each `gameState.grid.tile`, if `occupantType` is:

- `OccupantNone`: there should be nothing on that grid.
- `OccupantUnit`: render a unit there. Find the id in `gameState.units` for the unit details. Note that units with `hp` = 0 should not be rendered.
- `OccupantSoftBlock`: render a bomb there. Find the id in `gameState.softblocks` for the bomb details.
- `OccupantBomb`: render a bomb there. Find the id in `gameState.bombs` for the bomb details.

We expect `Occupant` must be found in either `gameState.units`, `gameState.softblocks`, `gameState.bombs`. If not, an error message should be displayed and the match shouldn't be proceeded.

### Logging via User Interactions

All occupants rendered in grid will be clickable in future. Add an event handler: When a occupant is clicked, log the message through `console.log()`. The message should contains `<occupantType> <occupantId> is clicked` followed by the properties of the clicked occupant. The real action implementation will be deferred to in future.

### Unit

Each `Unit` is rendered as a **32×32px** shape centered on its `Tile` using Phaser's `Graphics` API. Each should fill with color using `team` as index.

| `team` | Fill color | Hex        |
| ------ | ---------- | ---------- |
| 1      | Blue       | `0x212df3` |
| 2      | Red        | `0xf32d21` |

Archetype shape is drawn inside the fill (white stroke):

| Archetype (by `type` string) | Shape     |
| ---------------------------- | --------- |
| King                         | Star      |
| Fighter                      | Square    |
| Witch                        | Trigangle |
| Bandit                       | Circle    |

### SoftBlock

Each `SoftBlock` is rendered as a **42×42px** rounded rectangel centered on its `Tile` using Phaser's `Graphics` API.
`constants.ts`'s `SOFTBLOCK_COLORS` must match `0xe6e6e6`.

### Bomb

Each `Bomb` is drawn as a **24×24px** HEX `0x222222` circle centered on its `Tile`.
`countdown` is rendered as white text inside the circle.

## DevBootScene

At this stage, we only render the initialization of a Match. To ease the testing, update `GameCfg`:

- Set `stagePreset` to `MAP03` as it contains `Units`, `SoftBlocks`, `TerrainPlain` and `TerrainBlock`.
- Update both `p1Teams` and `p2Teams` to contain all King, Fighter, Witch and Bandit with below rules:
  - King must be the first unit.
  - Max size of the team is 5.

---

## Acceptance Criteria

1. Given a `GameState` with two teams, each `Unit` renders on the correct tile with the correct team color and archetype shape.
2. Given a `GameState` with bombs, each `Bomb` renders on the correct tile with the correct color and shape.
3. Given a `GameState` with softblocks, each `SoftBlock` renders on the correct tile with the correct color and shape.
4. Given `roomId` and `playerTokens` obtained from backend, the browser DevTool should display `roomId` and `playerTokens`.
5. Given a `Unit` shown in `grid`, When user clicks the `Unit`, the browser DevTool should display `Unit` details.
6. Given a `Bomb` shown in `grid`, When user clicks the `Bomb`, the browser DevTool should display `Bomb` details.
7. Given a `SoftBlocks` shown in `grid`, When user clicks the `SoftBlocks`, the browser DevTool should display `SoftBlocks` details.
