---
title: "Phase 3.1: Initialize Match Grid"
---

# Initialize Match Grid

## Context

Currently the basic requirement in backend is ready, but frontend is nothing. To start with, we must implement a frontend do display a Match.

## Goal

- Given a `GameState` json, `MatchScene` should render the Tiles according to `gameState.grid`.

## Non-Goal

- Rendering of activeTeam, units, bombs, softBlocks will be in next step.

## Layout

### Camera model

`MatchScene` uses a **fixed-scale viewport**: the camera pans over the world at a fixed zoom level. Tile size never changes during Phase 3.1.

The Phaser world camera (`this.cameras.main`) starts centered on the grid at **1× zoom**. The player sees whichever tiles fall within the viewport.

At `TILE_SIZE = 48px`, all supported grid sizes fit within the 1280×720 canvas at 1× zoom:

| Grid | World size | Fits 1280×720 |
|---|---|---|
| 7×7 | 336×336px | Yes |
| 11×11 | 528×528px | Yes |
| 15×15 | 720×720px | Yes |

For Phase 3.1 the camera is **static** — no pan or zoom. Camera pan via drag is introduced in `p3-spec002-match.md`. Zoom is deferred to a later spec.

### Tile size

Each `Tile` renders as a **48×48px** square. Defined in `constants.ts` as `TILE_SIZE`.

### Grid world position

The grid is drawn from world-space origin `(0, 0)`. Tile at `grid[row][col]` is at world position:

```
x = col * TILE_SIZE
y = row * TILE_SIZE
```

The camera is offset so the grid center aligns with the viewport center on scene entry:

```
cameraX = (grid[0].length * TILE_SIZE) / 2
cameraY = (grid.length    * TILE_SIZE) / 2
```

### Orientation

`MatchScene` targets **landscape orientation only**. On portrait mobile, a full-screen CSS overlay prompts "Rotate your device" and the canvas is hidden until landscape is detected. This is enforced via a CSS `@media (orientation: portrait)` rule — no game logic involved.

### HUD

HUD is out of scope for Phase 3.1. When introduced, it will use a separate **fixed camera** (does not pan with the world camera).

## Scene Entry

`MatchScene` is launched by `LoungeScene` after a successful `createMatch()`.

### Data assembled by LoungeScene before transition

| Field | Type | Source |
|---|---|---|
| `roomId` | `string` | `CreateMatchRoomResponse.id` |
| `playerTokens` | `[string, string]` | `CreateMatchResponse.playerTokens` |

### Transition

`LoungeScene` calls `this.scene.start('MatchScene', { roomId, playerTokens })`.
`LoungeScene` is stopped after the transition.

### MatchScene initialisation (create phase)

On `MatchScene.create(data)`:
1. Store `data.roomId` and `data.playerTokens` in scene fields.
2. Call `initRoom(data.roomId)` to configure the API client.
3. Call `getMatchState()` to fetch the initial `GameState` (no auth required).
4. Render the `grid` from the fetched `GameState`.

### Token usage rule

When submitting commands, `MatchScene` selects `playerTokens[GameState.activeTeam]` as the Bearer token.
Tokens remain valid until the match ends.

## Data Fetching

`MatchScene` calls `getMatchState()` once during `create()`.
No polling in Phase 3.1 — the grid is static until the next spec introduces turn submission.

## Terrain Visual Spec

Each `Tile` is rendered as a filled 48×48px rectangle using Phaser's `Graphics` API.
Hex values below are the source of truth; `constants.ts`'s `TERRAIN_COLORS` must match this table.

| `TerrainType`   | Fill   | Hex        | Behaviour (from engine)                                    |
|-----------------|--------|------------|--------------------------------------------------------------|
| `TerrainPlain`  | Green  | `0x4caf50` | Walkable by all units                                      |
| `TerrainBlock`  | Grey   | `0x9e9e9e` | Solid wall — not walkable, flyable, or jumpable            |
| `TerrainTower`  | Brown  | `0x795548` | High wall — not walkable, flyable, or jumpable             |
| `TerrainWater`  | Blue   | `0x2196f3` | Not walkable; flyable/jumpable; bombs disappear on contact |
| `TerrainLava`   | Orange | `0xff9800` | Not walkable; flyable/jumpable; bomb countdown forced to 1 |

All tiles share a **1px black border** to visually separate adjacent tiles.
Border color is defined as `TERRAIN_BORDER_COLOR` in `constants.ts`.

## Dev Bootstrap

Before scaffolding `DevBootScene`, delete the Vite template boilerplate that ships with `npm create vite`:

- `src/counter.ts`
- `src/style.css`
- `src/assets/vite.svg`
- `src/assets/typescript.svg`
- `src/assets/hero.png`

Replace `src/main.ts` with the Phaser game bootstrap (registered scenes, canvas config).

`LoungeScene` does not exist yet. To run `MatchScene` during development, scaffold a temporary `DevBootScene` that:

1. Calls `createMatchRoom()` to obtain a `roomId`.
2. Calls `createMatch()` with a hardcoded `CreateMatchRequest` (any valid stage preset).
3. Calls `this.scene.start('MatchScene', { roomId, playerTokens })`.

`DevBootScene` is registered in `main.ts` as the initial scene **only during Phase 3 development**. Remove it once `LoungeScene` is implemented.

## Acceptance Criteria

1. Given a `GameState` with an N×M `grid`, `MatchScene` renders N×M tiles each 48×48px at world position `(col * TILE_SIZE, row * TILE_SIZE)`.
2. Each tile's fill color matches `TERRAIN_COLORS[tile.type]`; a 1px black border is visible between all adjacent tiles.
3. On scene entry, the camera is centered on the grid regardless of grid size.
4. Given `getMatchState()` returns a network error, an error message is displayed — not a silent blank canvas.

## Log

Implementation issues found during the build (non spec gaps) are tracked in [`p3-spec001-match-log.md`](./p3-spec001-match-log.md).
