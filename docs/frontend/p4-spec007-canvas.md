---
title: "Phase 4.7: Auto Resize Canvas"
---

# Phase 4.7: Auto Resize Canvas

## Context

Phase 4.5 defines the 1280×720 canvas with x2 zoom. While it works great in a 4K screen, it fits badly in a smaller screen like MacBook or Phone. This specs adds a responsive layout so that it could resize to 640x360 with x1 zoom for smaller window size.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- To provide a smaller UI for smaller window size.

## Non-Goal

- To resize contiuously in response to the window size.
- Resize to values other than x1 or x2.

### Scale Manager

- The Base resolution stays at 640×360.
- Zoom should depend on the window size.
  - If window width is smaller than **1408px** or window height is smaller than **848px**, the Zoom should be `Phaser.Scale.NO_ZOOM`.
  - Otherwise, the Zoom should be `Phaser.Scale.ZOOM_2X`.
  - These thresholds carry a margin above the raw 1280×720 canvas size for breathing room; exact values may be tuned during implementation.
- `autoCenter: CENTER_BOTH` stays, centering the canvas within the window at whichever zoom is active.

---

## Acceptance Criteria

1. Given the game boots, when the canvas mounts, then its base resolution is 640×360 and it displays at a fixed 1280×720 (x2 zoom).
2. Given the browser window is resized, when the resize completes and window width is below 1408px or window height is below 848px, then the canvas display size changes to 640x360 (x1 zoom).
3. Given the canvas is displaying at x1 zoom, when the browser window is resized and completes at or above 1408×848, then the canvas display size changes back to 1280x720 (x2 zoom).
