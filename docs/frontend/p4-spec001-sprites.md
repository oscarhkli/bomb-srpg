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

No change from Phase 3.

---

## Layout

### Dimensions

- Instead of 1280×720 canvas, the canvas size will be adjusted to **640x320px** in future phase. This way the browser will scale the canvas up to HD or even 4K cleaner.
- Since we don't want to ruin the other Scenes that we won't touch at this stage, instead of resizing in current spec, create a second **640x320px** camera pointing at **(0, 0)** for `MatchScene`.
- `TILE_SIZE` was 48. Now it should be **32** so that the each tile size should now be **32x32px**.
  - All existing movement should adjust accordingly to accomodate the new tile size.
- `Unit` is targeted to be **40x32px**. They could be taller or shorter, but this is the optimial size.
- `Bombs` and `SoftBlocks` is targeted to be max **32x32px**. They could could be smaller.

> Note: Agents should tell whether I should do `MatchScene` Phaser camera ratio to be x2 here. Correct my term used here.

## Visual Spec

The new `MatchScene` is divided into 2 large regions, the left **480x320px** is `GameBoardRegion`, where the `Grid`, `Units`, `TileMap` sit. the right **160x320px** is `GameControlRegion`, where the `TurnPanel`, `TurnCommandPanel` and `MatchSummaryButton` sit.

### Sprites

Since the current sprites are still silhouettes, there isn't difference between idle, move, face left/right, etc. All sprites are exported to `web/public/assets/sprites/`. Source `.aseprite` files live outside the repo at `~/Aseprite/bomb-srpg/` (local, machine-specific, not committed).

Each source file's canvas (64×64) is intentionally larger than the visible character, to leave bleed-safe padding for shading. Visible bounds vary per unit and are **not** a fixed size — do not hardcode an anchor offset. Team color (`Blue`/`Red`) is stored as an Aseprite **tag** on each source file, not as separate files — there is no idle/walk/attack tag yet.

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

`Type: atlas (aseprite)` means loaded via `this.load.aseprite(key, pngPath, jsonPath)`, not `this.load.image`/`this.load.spritesheet` — the JSON carries per-frame trim data (`spriteSourceSize`), so the 64×64 padded canvas never leaks into gameplay positioning math.

All the other vector graphics not mentioned in above table should be retained, including the countdown of bombs.

### Grid

The Vector graphic `Grid` should be resize because `TILE_SIZE` is changed. It should not be drawn in the center of `GameBoardRegion`. It is expected that there are unused spaces in the 4 margins. Ignore them as this moment.

After the resizing and relocation of `Grid`, all the existing vector graphics and the replaced sprites should be resized and moved to new location accordingly. Bomb explosion rays and burning effect should also be adjusted as well.

> Note: Agent should correct my term of draw / render / other terms here for Vector Graphics and Sprites respectively. Update VISUAL_VOCAB.md when necessary.

### TurnCommandPanel

`TurnCommandPanel` should be resized to **144x144px**, placing at the bottom center of `GameControlRegion`, leaving **2px** bottom margin.

### ConfirmDialog

`ConfirmDialog` should be rendered at the center of the `MatchScene` camera instead of the center of canvas. Note that it will be change back to canvas once the resizing of the whole game is done.

### TurnPanel

`TurnPanel` should be put at the top center of `GameControlRegion`, leaving **2px** top margin.

### MatchSummaryPanel

`MatchSummaryPanel` should be rendered at the center of the `MatchScene` camera instead of the center of canvas. Note that it will be change back to canvas once the resizing of the whole game is done.

Resize the width of everything, including the `pillButton` to approx **80%** of the current size, nearest integer divisible by 4. Resize the height of everything, to approx **45%** of the current size, nearest integer divisible by 4. Reduce the font size inside `MatchSummaryPanel` by **4px**.

The target is to make sure all the elements are still fit in the resized `MatchSummaryPanel`. Since it will be eventually replaced by the sprites, the resizing doesn't have to be extremely accurate.

## Documentation

Update the obsolete part of `docs/design.md`, e.g., canvas size.

---

## Acceptance Criteria

1. Given … When … Then …
2. blast ray width and burn overlay remain proportionate to the 32px tile at the new scale
