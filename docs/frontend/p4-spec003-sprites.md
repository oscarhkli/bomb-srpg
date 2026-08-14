---
title: "Phase 4.3: Adopt Pixel Art Stage Backgrounds for MatchScene
---

# Phase 4.3: Adopt Pixel Art Stage Backgrounds for MatchScene

## Context

In Phase 3, `grid`'s terrain was rendered as colored vector rectangles (see `VISUAL_VOCAB.md` → Board). This spec replaces that rendering with the drafted pixel art Stage backgrounds for all 3 existing `StagePreset`s.

`grid` (`GameState.grid`, `Tile[][]`) itself is untouched — it stays the source of the board's dimensions (`setBoardOffset`) and occupant positioning. Only the per-tile vector drawing that used to visualize it is retired.

## Goal

- `MatchScene` displays the pixel-art Stage background matching the match's `StagePreset`.
- Retire the vector terrain rendering it replaces.

## Non-Goal

- Resize the Canvas of entire game - this should be done when all Scenes' resizings are done.
- A "toggle grid line" feature. A future spec may add a spatial-reference line overlay; it is a distinct feature from the terrain-color rendering retired here and is designed fresh when written.

## Scene Entry

No change from spec002.

---

## Layout

## Visual Spec

Unlike `Occupants`, Stage sprites are exported **trimmed**, but with the play area centered in the source canvas — the delivered art needs no padding compensation.

Sprites use origin **(0.5, 0.5)**, anchored at the canvas center.

| Entity           | Texture Key    | Path (relative to `sprites/`)       | Type             |
| ---------------- | -------------- | ----------------------------------- | ---------------- |
| Stage (Plain)    | stage_plain    | stages/Stage-Plain.png (+ .json)    | atlas (aseprite) |
| Stage (Standard) | stage_standard | stages/Stage-Standard.png (+ .json) | atlas (aseprite) |
| Stage (Divided)  | stage_divided  | stages/Stage-Divided.png (+ .json)  | atlas (aseprite) |

The active texture key is looked up by the match's `StagePreset.Name` (`gameCfg.stagePreset`), the same pattern `UNIT_TEXTURE_KEYS` uses for archetype+team (`p4-spec001-sprites.md`). These loads belong in `MatchScene.preload()`, added to the existing `SPRITE_MANIFEST`.

`grid`'s tile dimensions come from the match's `StagePreset`; the Stage background image may be a different size to allow decorative bleed beyond the playable area. The background and `grid` are center-aligned — not offset by a fixed pixel constant — so this holds regardless of a preset's dimensions or a background's canvas size.

The background renders at a lower depth than all occupants (Units, Bombs, SoftBlocks). Occupants remain separate sprites positioned per grid cell; the background carries no per-cell logic.

### Terrain Retirement

`renderGrid()`'s per-tile fill/stroke loop, and the `TERRAIN_COLORS`/`TERRAIN_BORDER_COLOR` constants it used, are dead code once the Stage background replaces them — delete both during implementation (same precedent as `p4-spec001-sprites.md`'s `UNIT_SIZE` retirement).

`DEPTH_GRID` is renamed `DEPTH_STAGE_BACKGROUND` — it now names one background `Image`'s depth, not a per-tile terrain layer.

## Documentation

Update `docs/design.md`'s Retro Art Strategy note (currently: "Tiles: Colored rectangles with borders") to reflect that Stage terrain now renders as a pixel-art background image, not procedural `Graphics`.

---

## Acceptance Criteria

1. Given `MatchScene` has loaded a match, when the scene renders, then the Stage background displays the pixel-art sprite matching the match's `StagePreset`, and no `TERRAIN_COLORS` fill/stroke renders.
2. Given a `StagePreset`'s `grid` dimensions or a Stage background's own canvas size, when the background renders, then it is center-aligned to `grid` regardless of either size.
3. Given occupants (Units, Bombs, SoftBlocks) are rendered, when the scene renders, then the Stage background renders behind all of them.
4. Given `MatchScene` is re-entered (new match or scene restart), when the scene renders, then the previous Stage background is destroyed and replaced, not left stacked or stale.
