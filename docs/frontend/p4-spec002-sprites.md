---
title: "Phase 4.2: Resize Height in MatchScene and Related Components"
---

# Phase 4.2: Resize Height in MatchScene and Related Components

## Context

Phase 4.1 intended to resize MatchScene to a smaller one for preparing pixel art adoption. However, the height of the scene was wrongly set. This spec fixes the scene height and updates any related changes related to the height.

## Goal

- Resize the height of the `MatchScene`.

## Non-Goal

- Not touching `TILE_SIZE`, `GameBoardRegion`/`GameControlRegion` widths, or the canvas resize.

## Scene Entry

MatchScene's `create()` adds the second 640×360 camera described in Layout → Dimensions as the scene's `main` camera (`makeMain: true`), removing the original full-canvas default camera. `MatchScene` renders through exactly one camera at a time — never both.

---

## Constants Updates

Update the following constants to **360**:
- MATCH_CAMERA_HEIGHT -> camera viewport (MatchScene.ts)
- GAME_BOARD_REGION_HEIGHT -> grid vertical centering (boardOffset.ts)
- GAME_CONTROL_REGION_HEIGHT -> TurnCommandPanel/MatchSummaryButton bottom-anchor math

## Documentation

Update the obsolete part of `docs/design.md`, e.g., the new camera dimensions.

---

## Acceptance Criteria

1. Given the second 640×360 camera is added for `MatchScene`, when `TitleScene` or `MatchSettingsScene` is visited, then their layout and clickable regions are unaffected (still full-canvas, pixel-identical to Phase 3).
2. Given a turn begins, a match ends, or sudden death triggers, when `TurnBanner`, `VictoryCutscene`, or `SuddenDeathCutscene` render, then each fills the new 640×360 camera viewport exactly (no clipping, no leftover space sized for the old 1280×720 canvas).
3. Given `GameBoardRegion` is now 480×360, when `Grid` renders, then it remains centered within the region with unused space split evenly across all 4 margins.
4. Given `GameControlRegion` is now 160×360, when `TurnCommandPanel` renders, then its 2px bottom margin is measured from the new 360px bottom edge, not the old 320px one.
5. Given `GameControlRegion` is now 160×360, when `MatchSummaryButton` renders, then it remains vertically centered in the gap between `TurnPanel`'s bottom edge and `TurnCommandPanel`'s top edge at the new height.
