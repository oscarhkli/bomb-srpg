---
title: "Phase 3.11: Title Scene"
---

# Title Scene

## Context

Entry scene when the app boots. Displays the game title and lets the Player choose how to start. The text-based `Title` is a placeholder; a sprite logo will replace it wholesale in a future polish spec.

## Goal

- Show game title.
- Show game mode options for selection.
- Player can enter `MatchSettingsScene` from `TitleScene`.

## Non-goal

- Polished art (including the sprite logo).
- Additional game modes (VS COM, online multiplayer).

## Scene Entry

`TitleScene` is the true entry scene: the app starts in `TitleScene`. Selecting a game mode option `fadeTransition`s to `MatchSettingsScene`.

Since `MatchSettingsScene` is no longer the entry scene and becomes exit-and-re-enterable:

- Re-entering `MatchSettingsScene` from `TitleScene` always starts with default settings; nothing is preserved from a previous visit.
- `MatchSettingsScene`'s async catalog load must not touch the scene after the Player has exited it (same hazard class as `p3-spec002-match-log` issue 1).

## Visual Effect

Sample as below:

```text
+----------------------+
|                      |
|     Bomb             |
|       Tactics        |
|                      |
|                      |
|     Start Game       |
|                      |
|                      |
|   {CopyrightText}    |
+----------------------+
```

`TitleScene` consists of 3 groups of components. All text uses `GAME_FONT_FAMILY`.

- `Title` is a composite component rendering at the top-center position, leaving **48px** from the top edge empty.
  - The words `Bomb` and `Tactics` are rendered in 2 lines, **48px** font size each.
  - Line 2 is indented: `T` starts at the same x-position as the `m` of `Bomb`.
- `GameModeSelectionPanel` sits at the center of `TitleScene`, listing all the options of the game mode.
  - Since currently there is only 1 game mode, render `Start Game` there.
  - The font size is **24px**.
- `CopyrightText` sits at the center bottom of `TitleScene`. Render `© {currentYear} Oscar oscarhkli.com`, leaving **12px** from the bottom edge empty.
  - The font size is **16px**.
  - Do not add hyperlink to it.

## User Interaction of TitleScene

- Add hover and click handlers for each option of `GameModeSelectionPanel` (only 1 at the moment).
- When hovered (Phaser `pointerover`):
  - Render a **24px** font size `💣` **24px** left of the option, representing the Player is considering that option.
- When the pointer leaves the option (Phaser `pointerout`):
  - Remove the `💣`.
- When clicked:
  - `fadeTransition` to `MatchSettingsScene`.
  - Once the transition starts, further clicks on any option are ignored.

## User Interaction of MatchSettingsScene

Since we now have `TitleScene`, when Player clicks `BackButton` in `UnitPage` for Player **1** in `MatchSettingsScene`, `MatchSettingsScene` should `fadeTransition` to `TitleScene`.

---

## Acceptance Criteria

1. Given Player accesses the app from scratch, When the app starts, Then Player should see `TitleScene`.
2. Given `TitleScene` is shown, When Player hovers a game mode option, Then `💣` renders left of it; When the pointer leaves the option, Then the `💣` disappears.
3. Given `TitleScene` is shown, When Player clicks `Start Game`, Then `TitleScene` should transition to `MatchSettingsScene`.
4. Given Player clicked `Start Game` and the transition has started, When Player clicks any option again, Then nothing additional happens.
5. Given Player sees `UnitPage` for Player 1, When Player clicks `BackButton`, Then `MatchSettingsScene` should transition to `TitleScene`.
6. Given `UnitPage` for Player 2, When Player clicks `BackButton`, Then it still goes to `UnitPage` for Player 1, not `TitleScene`.
7. Given Player returned to `TitleScene` from `MatchSettingsScene`, When Player clicks `Start Game`, Then `MatchSettingsScene` shows default settings.
8. Given `TitleScene` is shown, Then `CopyrightText` shows the current calendar year.
