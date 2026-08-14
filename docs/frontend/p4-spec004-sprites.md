---
title: "Phase 4.4: Swapping Sprites for Existing Vector Graphics in MatchSettingsScene"
---

# Phase 4.4: Swapping Sprites for Existing Vector Graphics in MatchSettingsScene

## Context

This spec deprecates the vector graphics of Units by replacing Archetypes sprites in `MatchSettingsScene`. This also redefines the canvas and object dimensions of the whole game to match with the retro style.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- `MatchSettingsScene` displays pixel art assets.
- Resize of the application screen.

## Non-Goal

- Resize the Canvas of entire game - this should be done when all Scenes' resizings are done.
- Complete Pixel Art adoption for `MatchSettingsScene` - some UI controls will still rely on vector drawings and emojis.

## Scene Entry

MatchSettingsScene's `create()` adds the second 640×360 camera described in Layout → Dimensions as the scene's `main` camera (`makeMain: true`), removing the original full-canvas default camera. `MatchSettingsScene` renders through exactly one camera at a time — never both.

---

## Layout

### Dimensions

- Instead of 1280×720 canvas, the canvas size will be adjusted to **640x360px** in future phase. This way the browser will scale the canvas up to HD or even 4K cleaner.
- Since we don't want to ruin the other Scenes that we won't touch at this stage, instead of resizing the canvas in this spec, mirror `p4-spec001-sprites.md`'s approach: add a second camera to `MatchSettingsScene` with a **640x360px** viewport positioned at **(0, 0)**, scrolled to world origin **(0, 0)**.

Camera stays at default zoom (1) — the 640×360 viewport needs no zoom multiplier.

## Visual Spec

### General Resizing

Halve every existing absolute layout constant in `MatchSettingsScene` and its child components (positions, sizes, spacing) to fit the new 640×360 viewport — e.g. `UNIT_SLOT_SIZE` 96px → 48px. Font sizes follow `p4-spec001-sprites.md`'s convention instead: reduce by a flat **4px**, not a percentage — e.g. `SETTINGS_TEXT_FONT_SIZE` 24px → 20px.

Sprites and other pixel-art assets scale down to fit their halved container. Non-pixel-perfect scaling is acceptable — current sprites are prototype silhouettes to be replaced by final pixel art in a later spec; this spec only needs the layout to function correctly at the new resolution.

Within `UnitCard`/`UnitSlot`, keep the sprite's `p4-spec001-sprites.md` origin `(0.5, 1)` and anchor its bottom edge to the container's bottom edge, rather than vertically centering it. Empty space above the sprite is acceptable at this stage.

### Sprite Use in UnitPage

All the Archetypes and Units rendering under `UnitPage` and the subsequent components should adopt pixel art sprites.

Since there are no portrait sprites for the archetypes, for fast prototyping, adopt the sprites from `MatchScene`. `UnitPage` renders before `MatchScene` in the normal flow, so `MatchSettingsScene` must guarantee unit sprite textures are loaded before `UnitPage` renders — it can't assume `MatchScene`'s texture cache is already populated.

Remove the team rounded square in `UnitCard` but keep the same spacing — the sprite's own team-colored silhouette (Blue/Red per archetype) already conveys team identity, making a separate colored background redundant. Remove the background color of `UnitSlot`. Instead, add a **2px** border with `TeamColor(team)`.

The spec of Unit stays unchanged as `p4-spec001-sprites.md`.

Note that this spec only replaces `Archetype` and `Unit` and does not replaces emojis, including `BombRange`.

### Retire Vector Graphic Rendering Features

Since the app doesn't use vector graphic drawing for Archetypes, some existing func and const can be retired.

- drawArchetypeIcon
- drawUnitSprite
- regularPolygonPoints
- starPoints

This list is not exhaustive — do a full code scan for final cleanup, including any tests asserting on this vector-drawing behavior and any constants that existed solely to parameterize these functions.

## Scene Exit

No explicit camera teardown needed: Phaser's `CameraManager` already destroys every camera on the scene's `shutdown` event, so re-entering `MatchSettingsScene` never stacks a duplicate.

---

## Acceptance Criteria

1. Given `MatchSettingsScene` has loaded the page, when the scene renders, then all Units display their pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
2. Given the second 640×360 camera is added for `MatchSettingsScene`, when `TitleScene` is visited, then their layout and clickable regions are unaffected (still full-canvas, pixel-identical to Phase 3).
3. Given `MatchSettingsScene`'s halved layout, when the scene renders at 640×360, then no `UnitPage`/`StagePage` content is clipped or overflows the camera viewport.
4. Given a `UnitCard` and an occupied `UnitSlot`, when they render, then `UnitCard` shows no background square behind its sprite, and `UnitSlot` shows a 2px `TeamColor(team)` border instead of a filled background.
5. Given a player reaches `MatchSettingsScene` without `MatchScene` ever having been visited in that session, when `UnitPage` renders, then unit sprites still display correctly (not blank/missing textures).

## Log

Implementation issues found during the build (non spec gaps) are tracked in [p4-spec004-sprites-log.md](./p4-spec004-sprites-log.md).
