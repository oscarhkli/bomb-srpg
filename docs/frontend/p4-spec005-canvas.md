---
title: "Phase 4.5: Resize Canvas to 640x360"
---

# Phase 4.5: Resize Canvas to 640x360

## Context

This spec finishes the canvas resize migration. `TitleScene` is the last scene still on the 1280×720 layout, so there is no remaining scene that needs the canvas to stay full-size. The temporary "add a second 640×360 camera, remove the default one" step used by `MatchScene` and `MatchSettingsScene` during their pixel-art specs (p4-spec001 – p4-spec004) is no longer needed for `TitleScene` and can be retired everywhere else in this same spec.

This is also a deliberate switch away from the game's current responsive behavior (`Phaser.Scale.FIT`, which scales the canvas to fill the browser window). The canvas becomes a fixed 1280×720 CSS px regardless of window size — see Scale Manager below.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- Resize `TitleScene`'s layout to fit a 640×360 viewport.
- Resize the actual `Phaser.Game` canvas from 1280×720 to 640×360.
- Retire the temporary second camera in `MatchScene` and `MatchSettingsScene`, restoring each to Phaser's single default main camera now that it already matches the 640×360 canvas.

## Non-Goal

- Complete Pixel Art adoption for the whole game — some UI controls still rely on vector drawings and emojis.
- Resizing `ErrorPanel` (`ERROR_PANEL_X`/`Y`/`WIDTH`/`HEIGHT`) to fit the 640×360 camera. It stays out-of-viewport post-resize, same as when p4-spec001 first flagged it — presentation for dev/debug-only surfacing needs its own design discussion before it's worth coding.

## Scene Entry

- **Game config:** canvas base dimensions change from 1280×720 to 640×360, displayed at a fixed **x2 zoom** (see Scale Manager below).
- **`TitleScene`:** renders through Phaser's single default main camera, sized to the canvas. No second camera is added — unlike the other scenes, `TitleScene` never went through the transition period, so there's no scaffolding to add or remove.
- **`MatchScene` / `MatchSettingsScene`:** drop the `cameras.add(...)` / `cameras.remove(defaultCamera)` pair added in their pixel-art specs. Each scene goes back to using `this.cameras.main` untouched, since the default camera now already covers the 640×360 canvas.

### Scale Manager

- **Base resolution:** 640×360 — unchanged by zoom, this is what all camera and layout math targets.
- **Scale mode:** `Phaser.Scale.NONE` — the canvas display size is fixed, not recalculated on browser resize.
- **Zoom:** fixed **x2** (`Phaser.Scale.ZOOM_2X`), so the canvas always renders at 1280×720 CSS px regardless of window size.
- **`pixelArt: true`** on the game config, so the x2 CSS scale stays crisp (nearest-neighbor) instead of blurring sprites.
- `autoCenter: CENTER_BOTH` stays, centering the fixed 1280×720 canvas within the window.

---

## Layout

### Dimensions

- Canvas: 1280×720 → **640×360**.
- `TitleScene`: halve every existing absolute layout constant (positions, sizes, spacing) — e.g. `UNIT_SLOT_SIZE` 96px → 48px. Font sizes follow `p4-spec001-sprites.md`'s convention instead: reduce by a flat **4px**, not a percentage — e.g. `TITLE_FONT_SIZE` 48px → 44px.
- `MatchScene` / `MatchSettingsScene`: no layout change — their constants were already halved to 640×360 when the temporary camera was added.

## Visual Spec

### TitleScene Resizing

Halve every existing absolute layout constant in `TitleScene` and its child components per the Dimensions section above.

### Camera Cleanup (MatchScene, MatchSettingsScene)

Remove the temporary `cameras.add(...)`/`cameras.remove(defaultCamera)` calls and the now-unused viewport constants (`MATCH_CAMERA_WIDTH`/`MATCH_CAMERA_HEIGHT` or equivalent) that sized the second camera. No behavior change — the default main camera already renders at 640×360 once the canvas resizes.

## Scene Exit

No explicit camera teardown needed: Phaser's `CameraManager` already destroys every camera on the scene's `shutdown` event, so re-entering any scene never stacks a duplicate.

---

## Acceptance Criteria

1. Given the game boots, when the canvas mounts, then its base resolution is 640×360 and it displays at a fixed 1280×720 (x2 zoom).
2. Given the browser window is resized, when the resize completes, then the canvas display size does not change.
3. Given `TitleScene` has loaded, when the scene renders, then every layout constant (position, size, spacing) is exactly half its pre-resize value and every font size is its pre-resize value minus 4px, with no clipped or overflowing content.
4. Given `MatchScene` or `MatchSettingsScene` is entered after this change, when the scene creates, then it renders through a single main camera (no `cameras.add` call), and layout is unaffected versus the prior temporary-camera version.
5. Given any scene is re-entered (e.g. Title → Match → Title), when `create()` runs again, then no duplicate or stale cameras accumulate.
