---
title: "Phase 5.1: 2.5D Isometric Grid"
---

# 2.5D Isometric Grid

## Context

Phase 3 uses a flat top-down 2D grid. This spec replaces it with a 2.5D isometric projection — rectangular tiles with a horizontal tilt — for a more tactical-SRPG feel.

## Goal

_TODO: fill in_

## Non-Goal

- Unit/bomb visual redesign for isometric perspective.
- Elevation (height differences between terrain types).

## Scene Entry

### Prerequisites

- Phase 3 grid, unit, and bomb rendering complete.
- `GridRenderer` refactored to accept a coordinate-mapping strategy (so flat 2D → isometric swap is isolated).

---

## Layout

This is a significant rendering change. It affects:
- Tile shape: from square to diamond (isometric rhombus)
- Coordinate mapping: screen position from `(col, row)` changes to staggered isometric formula
- Depth sorting: objects further from the camera (higher row) must render behind closer objects
- Click-to-tile math: world coordinates must account for the isometric transform

Notes:
- Phaser 4 does not have native isometric support. Coordinate math must be implemented manually.
- Isometric tile width : height ratio is typically 2:1. At `TILE_SIZE = 48px`, tile width = 96px, tile height = 48px.
- Depth sorting: draw tiles in row-major order (row 0 first, last row on top).

---

## Acceptance Criteria

_TODO: fill in_
