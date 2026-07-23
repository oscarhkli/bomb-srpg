---
title: "Phase 3.11: Title Scene — Implementation Log"
---

# Implementation Log: p3-spec011-title

## Issue 1: Phaser retains previous scene data when `scene.start()` gets no data argument

- **Status:** Solved
- **Found by:** frontend-code-review spec-conformance pass (AC 7).
- **What happened:** `TitleScene` originally called `this.scene.start('MatchSettingsScene')` with no data, assuming "pass nothing = clean entry". Phaser's `Systems.start` only overwrites `settings.data` when a data argument is provided, so the `{ gameCfg }` that `MatchScene`'s Return-to-Settings passed earlier survives and reaches `create()` on the next entry. The flow Title → Settings → Match → Return to Settings → Back → Title → Start Game would show the previous match's settings, violating AC 7.
- **Fix:** `TitleScene` passes an explicit empty object — `this.scene.start('MatchSettingsScene', {})` — forcing Phaser to drop the stale data. The AC 7 test asserts the `{}` argument so a bare `start()` regresses the test.

## Issue 2: Game font could rasterize as the browser fallback on first paint

- **Status:** Solved
- **Found by:** frontend-code-review correctness pass.
- **What happened:** Roboto was loaded via a Google Fonts `<link>` stylesheet (per p3-spec003), but stylesheet fonts are fetched lazily on first DOM use. No DOM text uses Roboto and canvas rendering doesn't trigger the lazy fetch, so `TitleScene` — the first paint after boot — could rasterize its `Text` objects (including the measured "Bo" indent) permanently in the fallback font.
- **Fix:** Self-hosted `roboto-400.woff2` in `web/public/fonts/`, loaded in `TitleScene.preload()` via `this.load.font()`; the Google Fonts `<link>`/`preconnect` tags were removed from `index.html`. The spec's new "Font Loading" section records this as superseding p3-spec003's mechanism. Weight 700 is not vendored — nothing in `web/src` uses bold.
