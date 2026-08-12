---
title: "Phase 4.1: Swapping Sprites for Existing Vector Graphics in MatchScene"
---

# Phase 4.1: Swapping Sprites for Existing Vector Graphics in MatchScene

## Context

The frontend of Phase 3 is a POC version of a frontend, with dummy Vector Graphics. The final products is targeted to be a pixel art game. Phase 4 replaces vector graphics for pixel art assets.

Phase 4.1 dummy, still silhouettes adopts Units, Bombs and SoftBlocks in `MatchScene`. This also redefines the canvas and object dimensions of the whole game to match with the retro style.

## Goal

- `MatchScene` displays pixel art assets of Units, Bombs, SoftBlocks.
- Resize of the application screen.
- Resize Bomb explosion rays and burning effect to align with the resize of the game objects.

## Non-Goal

- Resize the Canvas of entire game - this should be done when all Scenes' resizings are done.
- Adopting Tile Map sprites - defer to next phase after Units, Bombs, SoftBlocks and `MatchScene` resizing are settled.
- Abandoning Vector Graphics for Scenes other than `MatchScene`.
- Abandoning UI, Controls other than Units, Bombs and SoftBlocks.

## Scene Entry

MatchScene's `create()` adds the second 640×320 camera described in Layout → Dimensions as the scene's `main` camera (`makeMain: true`), removing the original full-canvas default camera. `MatchScene` renders through exactly one camera at a time — never both.

---

## Layout

### Dimensions

- Instead of 1280×720 canvas, the canvas size will be adjusted to **640x320px** in future phase. This way the browser will scale the canvas up to HD or even 4K cleaner.
- Since we don't want to ruin the other Scenes that we won't touch at this stage, instead of resizing in current spec, create a second camera for `MatchScene` with a **640x320px** viewport positioned at **(0, 0)** on the canvas, scrolled to world origin **(0, 0)**.
- `TILE_SIZE` was 48. Now it should be **32** so that each tile size should now be **32x32px**.
  - All existing movement should adjust accordingly to accommodate the new tile size.
- `Unit` art is budgeted at roughly **40x32px** visible content — an authoring target, not a runtime size; sprites render at their native trimmed size (see Visual Spec → Sprites).
- `Bombs` and `SoftBlocks` art is budgeted at max **32x32px** visible content, same authoring-only caveat.
- `UNIT_SIZE` (`constants.ts`) becomes dead code once Units switch to sprites — it's only used in the vector-drawing code this spec removes. Delete it during implementation.

Camera stays at default zoom (1) — the 640×320 viewport already matches `GameBoardRegion` (480×320) + `GameControlRegion` (160×320) 1:1, so no zoom multiplier is needed.

## Visual Spec

The new `MatchScene` is divided into 2 large regions, the left **480x320px** is `GameBoardRegion`, where the `Grid` and `Units` sit. The right **160x320px** is `GameControlRegion`, where the `TurnPanel`, `TurnCommandPanel` and `MatchSummaryButton` sit.

### Sprites

Since the current sprites are still silhouettes, there isn't difference between idle, move, face left/right, etc. All sprites are exported to `web/public/assets/sprites/`. Source `.aseprite` files live outside the repo at `~/Aseprite/bomb-srpg/` (local, machine-specific, not committed).

Each source file's canvas (64×64) is intentionally larger than the visible character: a fixed 16px band at the bottom is reserved for bleed-safe padding and optional visual flourishes (e.g. a weapon tip extending past the feet); the remaining space above is unused margin and varies per unit.

Sprites are exported **untrimmed** — the full 64×64 canvas is delivered as-is for every entity in the table below.

Sprites use origin **(0.5, 1)** — horizontally centered, vertically anchored to the bottom of the untrimmed 64×64 canvas. Since that canvas always reserves a fixed 16px band at the bottom (`SPRITE_GROUND_MARGIN`), position each sprite so its anchor sits `SPRITE_GROUND_MARGIN` below the tile's bottom edge — this compensates for the reserved band uniformly across all units, so every sprite's feet rest on the tile floor regardless of its individual visible height.

| Entity         | Texture Key       | Path (relative to `sprites/`)    | Type             |
| -------------- | ----------------- | -------------------------------- | ---------------- |
| Fighter (Blue) | unit_fighter_blue | units/Fighter-Blue.png (+ .json) | atlas (aseprite) |
| Fighter (Red)  | unit_fighter_red  | units/Fighter-Red.png (+ .json)  | atlas (aseprite) |
| King (Blue)    | unit_king_blue    | units/King-Blue.png (+ .json)    | atlas (aseprite) |
| King (Red)     | unit_king_red     | units/King-Red.png (+ .json)     | atlas (aseprite) |
| Bandit (Blue)  | unit_bandit_blue  | units/Bandit-Blue.png (+ .json)  | atlas (aseprite) |
| Bandit (Red)   | unit_bandit_red   | units/Bandit-Red.png (+ .json)   | atlas (aseprite) |
| Witch (Blue)   | unit_witch_blue   | units/Witch-Blue.png (+ .json)   | atlas (aseprite) |
| Witch (Red)    | unit_witch_red    | units/Witch-Red.png (+ .json)    | atlas (aseprite) |
| Bomb           | bomb              | Bomb.png (+ .json)               | atlas (aseprite) |
| SoftBlock      | soft_block        | SoftBlock.png (+ .json)          | atlas (aseprite) |

`Type: atlas (aseprite)` means loaded via `this.load.aseprite(key, pngPath, jsonPath)`, not `this.load.image`/`this.load.spritesheet` — this matches Aseprite's export format (a PNG + a JSON sprite-sheet description), independent of trim. Since sprites are untrimmed, `spriteSourceSize` always equals the full 64×64 canvas; positioning relies on the fixed `SPRITE_GROUND_MARGIN` convention, not per-frame trim data. These loads belong in `MatchScene`'s `preload()` (it doesn't currently have one) — `MatchScene` is the only scene that needs these textures.

Sprites render at their native (untrimmed, 64×64) size — no runtime scaling. The `40×32px`/`32×32px` figures above describe the visible-content budget within that canvas, not a `setDisplaySize` requirement.

All the other vector graphics not mentioned in above table should be retained, including the countdown of bombs.

The Bomb sprite replaces only the 💣 glyph inside the existing Bomb `Container`; the countdown text child is retained unchanged.

> Note: Units and SoftBlocks change from drawn `Graphics` shapes to textured `Sprite`s. The shared render-state tracking them is read and mutated by more than just the code that first draws them (e.g. turn-resolution and movement-animation code) — plan for updates beyond the initial render function.

### Grid

The Vector graphic `Grid` should be resize because `TILE_SIZE` is changed. It should be centered within `GameBoardRegion`, with unused space split evenly across all 4 margins.

After the resizing and relocation of `Grid`, all the existing vector graphics and the replaced sprites should be resized and moved to new location accordingly. Bomb explosion rays and burning effect should also be adjusted as well.

### Occupant Depth

Since `Unit` sprites are taller than one tile, a unit standing on a lower row can visually overlap the sprite of an occupant on the row above it. Depth must reflect row position: an occupant on a lower row (larger Y) renders in front of one on a higher row (smaller Y) wherever they overlap. This applies uniformly across Units, SoftBlocks, and Bombs, and must be re-evaluated whenever an occupant's row changes (e.g. after a move), not just at initial render.

### TurnCommandPanel

`TurnCommandPanel` should be resized to **144x144px**, placing at the bottom center of `GameControlRegion`, leaving **2px** bottom margin.

### ConfirmDialog

`ConfirmDialog` should be rendered at the center of the `MatchScene` camera instead of the center of canvas. Note that it will be changed back to canvas once the resizing of the whole game is done.

### TurnPanel

`TurnPanel` should be put at the top center of `GameControlRegion`, leaving **2px** top margin.

### MatchSummaryPanel

`MatchSummaryPanel` should be rendered at the center of the `MatchScene` camera instead of the center of canvas. Note that it will be changed back to canvas once the resizing of the whole game is done.

Resize the width of everything, including the `pillButton` to approx **80%** of the current size, nearest integer divisible by 4. Resize the height of everything, to approx **45%** of the current size, nearest integer divisible by 4. Reduce the font size inside `MatchSummaryPanel` by **4px**.

The target is to make sure all the elements are still fit in the resized `MatchSummaryPanel`. Since it will be eventually replaced by the sprites, the resizing doesn't have to be extremely accurate.

### Full-Screen Overlays

`TurnBanner`, `VictoryCutscene`, and `SuddenDeathCutscene` all derive their layout purely from `this.scene.cameras.main.width`/`height` — no hardcoded canvas dimensions. Since `cameras.main` now resolves to the new 640×320 camera (see Scene Entry), all three fill the new viewport correctly with no code changes.

`VictoryCutscene`'s fixed-pixel constants (banner height, title/subtitle font sizes, button dimensions) don't shrink automatically — apply the same resize recipe as `MatchSummaryPanel`: ~80% width / ~45% height (nearest multiple of 4), font sizes reduced by 4px. `VictoryCutscene` and `TurnBanner` share `TURN_BANNER_HEIGHT`/`TURN_BANNER_TEXT_COLOR` — resizing that constant affects both.

`ErrorPanel` is explicitly out of scope for this spec — it positions itself with fixed pixel constants (`ERROR_PANEL_X`/`Y`/`WIDTH`/`HEIGHT`), not `cameras.main`, so it is *not* automatically fixed by the camera swap and will render clipped/off-viewport until a follow-up spec resizes it.

## Scene Exit

On shutdown (leaving `MatchScene` — match ends, or the player backs out), remove the camera added in Scene Entry before the scene restarts for a new match. Otherwise re-entering `MatchScene` leaves a stale camera reference and/or stacks a duplicate camera each time.

## Documentation

Update the obsolete part of `docs/design.md`, e.g., `TILE_SIZE` and the new camera dimensions.

---

## Acceptance Criteria

1. Given `MatchScene` has loaded a match, when the scene renders, then all Units, Bombs, and SoftBlocks display their pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
2. Given `TILE_SIZE` is now 32px, when the Grid and any occupant render, then tile boundaries, occupant positions, and click hit-areas align to the new 32px grid with no leftover 48px-derived offsets.
3. Given a unit sprite is positioned on a tile, when it renders, then its origin-anchor is offset by `SPRITE_GROUND_MARGIN` below the tile's bottom edge so the visible character's feet rest on the tile floor, regardless of that unit's individual sprite height.
4. Given the second 640×320 camera is added for `MatchScene`, when `TitleScene` or `MatchSettingsScene` is visited, then their layout and clickable regions are unaffected (still full-canvas, pixel-identical to Phase 3).
5. Given `TurnCommandPanel`, `TurnPanel`, `ConfirmDialog`, and `MatchSummaryPanel` are repositioned per this spec's Visual Spec section, then all remain fully clickable within their specified region/camera-center.
6. Given a bomb explodes, when the blast rays and burn overlay render at the new 32px tile scale, then their size remains visually proportionate to the tile.
7. Given a turn begins, a match ends, or sudden death triggers, when `TurnBanner`, `VictoryCutscene`, or `SuddenDeathCutscene` render, then each fills the new 640×320 camera viewport exactly (no clipping, no leftover space sized for the old 1280×720 canvas).
8. Given a Unit, SoftBlock, or Bomb is rendered on the board, when the player clicks its tile, then the corresponding click handler fires — matching Phase 3 behavior, not just the Bomb-only case.
9. Given two occupants on adjacent rows whose sprites visually overlap, when both render, then the occupant on the lower row (larger Y) is fully visible and the one above it is partially hidden behind it.
