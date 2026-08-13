---
title: "Phase 4.3: Adopt Dual Grid System for MatchScene Tile Map
---

# Phase 4.3: Adopt Dual Grid System for MatchScene Tile Map

## Context

In Phase 3 we introduced `grid` which is the vector graphical representation. This spec introduces the drafted pixel art sprites of all 3 existing stages. This also adopts [dual-grid tilemap](https://www.lexaloffle.com/bbs/?tid=143710) mechanism.

## Goal

- `MatchScene` displays pixel art assets of `grid` of all 3 stages.

## Non-Goal

- Resize the Canvas of entire game - this should be done when all Scenes' resizings are done.

## Scene Entry

No change from spec002.

---

## Layout

## Visual Spec

Unlike `Occupants`, Stage sprites are exported **trimmed** — the full canvas is delivered for every entity in the table below.

Sprites use origin **(0.5, 0.5)** — centered anchored of the canvas.

| Entity           | Texture Key    | Path (relative to `sprites/`)       | Type             |
| ---------------- | -------------- | ----------------------------------- | ---------------- |
| Stage (Plain)    | stage_plain    | stages/Stage-Plain.png (+ .json)    | atlas (aseprite) |
| Stage (Standard) | stage_standard | stages/Stage-Standard.png (+ .json) | atlas (aseprite) |
| Stage (Divided)  | stage_divided  | stages/Stage-Divided.png (+ .json)  | atlas (aseprite) |

We adopt dual-grid tilemap. Splitting grid into visual layer and logical layer. Since logical layer stays 9x9x32 square pixels as of today but visual layer could be larger, on design perspective this visual layer, so both layer should align at center instead of using a hardcoded 16px offset.

> Note: Agent should correct the naming of visual layer and logical layer.

The visual layer should be render slightly higher depth than the logical layer, so that the visual layer can hide the logical layer but not the other images.

> Note: Agent should analyze if we should not render the logical layer at all.

---

## Acceptance Criteria

1. Given `MatchScene` has loaded a match, when the scene renders, then grid display its pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
