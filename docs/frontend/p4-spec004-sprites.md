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
- Since we don't want to ruin the other Scenes that we won't touch at this stage, instead of resizing in current spec, create a second camera for `MatchScene` with a **640x360px** viewport positioned at **(0, 0)** on the canvas, scrolled to world origin **(0, 0)**.
- Draw a border line with width 4px on both `MatchScene` and `MatchSettingsScene` to help visualizing the **640x360px** viewport. Note that this is a temp component and will be remove once the canvas resizing is finished.

Camera stays at default zoom (1) — the 640×360 viewport - no zoom multiplier is needed.

## Visual Spec

### General Resizing

Resize and relocate **everything** in `MatchSettingScene` and its child components to `50%`.

> Note: Agent should tell if I have to explicitly spec every components how we should do the resize. Should we refer to PR #41?

### Sprites Use in UnitPage

All the Archetypes and Units rendering under `UnitPage` and the subsequent components should adopt pixel art sprites.

Since there is no portrait sprites for the archetypes, for fast prototyping, adopt the sprites from `MatchScene`. Remove the the team rounded square in `UnitCard` but keep the same spacing - Sprite has its own background color. Remove the background color of `UnitSlot`. Instead, add a **2px** border with `TeamColor`.

The spec of Unit stays unchanged as `p4-spec001-sprites.md`.

Note that this spec only replaces `Archetype` and `Unit` and does not replaces emojis, including `BombRange`.

### Retire Vector Graphic Rendering Features

Since the app doesn't use vector graphic drawing for Archtypes, some existing func and const can be retired.

- drawArchetypeIcon
- drawUnitSprite
- regularPolygonPoints
- starPoints

Do a code scan for final cleanup.

## Scene Exit

On shutdown (leaving `MatchSettingsScene` — match ends, or the player backs out), remove the camera added in Scene Entry before the scene restarts for a new match. Otherwise re-entering `MatchSettingsScene` leaves a stale camera reference and/or stacks a duplicate camera each time.

---

## Acceptance Criteria

1. Given `MatchSettingsScene` has loaded the page, when the scene renders, then all Units display their pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
2. Given the second 640×360 camera is added for `MatchSettingsScene`, when `TitleScene` is visited, then their layout and clickable regions are unaffected (still full-canvas, pixel-identical to Phase 3).
