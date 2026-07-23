---
title: "Phase 3.10: Match Settings Scene - Stage Page"
---

# Match Settings Scene - Stage Page

## Context

Phase 3.9 introduced Match Settings Scene with `UnitPage`. This spec renders the body region of `StagePage` and the event handlers to complete the flow of Match Settings, allowing Player to configure and start a match.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- Render `StagePage` in `MatchSettingsScene` to allow the Player to configure the match.
- Enter `MatchScene` via `MatchSettingsScene`.
- Start the app from `MatchSettingsScene`.

## Non-Goal

- Polished animations or tweens.
- VS COM or Online multi-player mode of `MatchSettingsScene`.

## Scene Entry

Starting from Phase 3.10, `DevBootScene` is no longer necessary. The app should start in `MatchSettingsScene`.

`MatchScene` is launched by `MatchSettingsScene` after a successful `createMatch()`.

Every Scene transition (not just `Page` swaps within `MatchSettingsScene`) is a `fadeTransition`: the outgoing Scene `fadeOut`s before calling `scene.start()`/`scene.restart()`, and the incoming Scene `fadeIn`s unconditionally in its own `create()`. Both halves of the pair are required — a Scene that fades out without its destination fading back in leaves the camera stuck dark. This applies to `MatchSettingsScene -> MatchScene` (`StartMatchButton`, below) and `MatchScene -> MatchSettingsScene` (Return to Settings).

## StagePage

The main region of `StagePage` consists of 2 panels, `StagesPanel` and `StageDetailPanel`, sitting in 2 columns. `StagePage` receives `stagePresets` (from `MatchSettingsScene`'s catalog) and the shared `gameCfg` reference, mirroring how `UnitPage` already receives `archetypes`/`gameCfg`.

### Visual Effect of Stages Panel

`StagesPanel` covers 60% of `StagePage`. Each side should have **12px** padding.

`StagesPanel` lists all `StageCards` in a row, center aligned. Each card should have **12px** spacing. Note that in future there will be more than 8 stagePresets available, so `StagesPanel` should be horizontally swipable. Since there is not enough StagePresets to scroll at the moment, the swipe effect is a **non-goal** unless it's very simple to do.

#### Stage Card

`StageCard` is a **160Wx160Hpx** rounded rectangle container, with **12px** padding on 4 side. [User Interaction](#user-interaction-and-visual-effect-of-stage-page) will describe a **Select** behavior. When a `StageCard` is selected, add a border of **4px** in HEX `0xdc9e23`.

- For simplicity, `stagePreset.name` should be rendered at the center of the card, with font size **36px**.
- If `MatchSettingsScene.gameCfg` contains `stagePreset`, select it according. Otherwise, select the first `StageCard`.

### Visual Effect of StageDetail Panel

`StagesDetailPanel` covers 40% of `StagePage`. Reserve a rectangular region of size 80% of `StagesDetailPanel`, namely `InnerPanel`, for the 4 rows below — `InnerPanel` is a layout region, not a drawn/visible box. Each side of `InnerPanel` should have **12px** padding.

Similar to `ArchetypesPanel` in `UnitPage`, `StagesDetailPanel` lists the selected `StageCard`'s `stagePreset` details.

Inside `InnerPanel`, render 4 rows of text using `stagePreset.name`, `stagePreset.description`, `stagePreset.width`, `stagePreset.height` and `stagePreset.maxTurns`. Each row should have **12px** space. Sample is shown below:

```text
+--------------------+
|      {name}        |
| {description}      |
| {description-wrap} |
| {width} x {height} |
|  ❰  {maxTurns}  ❱  |
+--------------------+
```

- Font size should all be **24px**.
- `{name}`: Center aligned.
- `{description}`: Center aligned with 1 extra line reserved for line wrapping.
- `{width} x {height}`: Center aligned. `x` should be at the dead center.
- `{maxTurns}`: This is a selection component named `MaxTurnsSelector`, with a `❰` and `❱` wrapped.
  - 2 arrows symbol should be at a fixed position, i.e. **24px** at the nearest edge. The font color is HEX `0xdc9e23`.
    - This 24px is measured from `InnerPanel`'s own edge, not from the padded content edge (i.e. it sits 12px inside the row's own 12px content padding, not flush with it).

### User Interaction and Visual Effect of Stage Page

- All `StageCards` in `StagesPanel` have click handlers.
  - If a `StageCard` is clicked, the `StageCard` is selected. The other `StageCard` should be unselected.
  - `StagesDetailPanel` should get the infomation from `stagePresets` and replace all the information inside.
  - `maxTurns` is a cyclic selection component (`MaxTurnsSelector`). The available selections are: `💣`, `15`, `20`, `30`, `45`, `60`. Note that `💣` means `0` in behind.
    - Selecting *any* `StageCard` — including re-selecting one whose `maxTurns` was previously customized — always resets `MaxTurnsSelector` to `stagePreset.maxTurns` (that stage's recommended value), discarding any prior customization for that stage. Unlike the other components, it never restores the value from `MatchSettingsScene.gameCfg`.
      - This applies on `StagePage`'s initial mount too, not just on click: the selector always starts at the selected `StageCard`'s `stagePreset.maxTurns`, even if `MatchSettingsScene.gameCfg.maxTurns` holds a previously customized value (e.g. from a prior visit to this page). Only the selected `StageCard` itself is restored from `gameCfg` (via `gameCfg.stagePreset`) on mount.
    - Add a click handler for `❰` - clicking this will shift the selection left, e.g., `20 -> 15`. Note that it's cyclic, i.e., `💣 -> 60`.
    - Add a click handler for `❱` - clicking this will shift the selection right, e.g., `15 -> 20`. Note that it's cyclic, i.e., `60 -> 💣`.
    - If the currently selected value is `stagePreset.maxTurns`, render a `🌟` with font size **16px** next to the value, meaning it's recommended max turns. Note that it should be a fix position - never move any elements to accommodate it.
  - Any user interaction in `StagePage` should update `MatchSettingsScene.gameCfg.stagePreset` and `MatchSettingsScene.gameCfg.maxTurns` immediately.
- `StartMatchButton`:
  - Since 1 and only 1 `StageCards` is always selected, validation isn't necessary in `StagePage`, i.e., `StartMatchButton` should always enabled.
  - If Player clicks `StartMatchButton`, follow how `DevBootScene` calls `create()`:
    - Call `createMatchRoom()` to get `roomId`.
    - Use `roomId` to `initRoom()`.
    - Use `roomId` and `MatchSettingsScene.gameCfg` to call `createMatch()`.
    - `fadeTransition` to `MatchScene` with `roomId` and `playerTokens`.
  - If `createMatchRoom()` or `createMatch()` fails, report the error and stay on `StagePage` (no transition to `MatchScene`).

---

## Acceptance Criteria

1. Given both Player 1 and Player 2 configure their `Units` in `UnitPage`, and they select the `Stage` in `StagePage`, When Players click `StartMatchButton`, the game creates the `Room` and `Match` and transitions to `MatchScene`.
2. Given `MatchSettingsScene` boots, When no `DevBootScene` precedes it, Then the app starts directly in `MatchSettingsScene` (no dev-boot scaffolding in the scene list).
3. Given `StagePage` mounts, Then it renders one `StageCard` per entry in `stagePresets`, each labeled with `stagePreset.name`.
4. Given `MatchSettingsScene.gameCfg.stagePreset` matches one of the `stagePresets`, When `StagePage` mounts, Then the matching `StageCard` is selected; otherwise the first `StageCard` is selected.
5. Given a `StageCard` is selected, When Player clicks a different `StageCard`, Then the clicked card becomes selected (bordered), the previously selected card is unselected, and `StageDetailPanel` immediately replaces its content with the newly selected preset's details.
6. Given any `StageCard` is clicked (including re-clicking the already-selected one), Then `MaxTurnsSelector` resets to that `stagePreset.maxTurns`, discarding any prior customization — this includes `StagePage`'s initial mount, which always seeds from the selected preset's `maxTurns` rather than from `gameCfg.maxTurns`.
7. Given `MaxTurnsSelector` is showing some value, When Player clicks `❰` or `❱`, Then the value cycles to the previous/next entry in `[💣, 15, 20, 30, 45, 60]`, wrapping at both ends (`💣 -> 60` on left from `💣`, `60 -> 💣` on right from `60`), and `MatchSettingsScene.gameCfg.maxTurns` is committed immediately.
8. Given `MaxTurnsSelector`'s current value equals the selected preset's `maxTurns`, Then the 🌟 recommended glyph is shown next to it; otherwise it is hidden, without shifting any neighboring element's position.
9. Given `StagePage` is currently active, `StartMatchButton` is always enabled (never rendered disabled), since a `StageCard` is always selected.
10. Given `StagePage` is currently active, When Player clicks `StartMatchButton`, Then the scene calls `createMatchRoom()`, `initRoom()`, `createMatch()` with `gameCfg` in sequence and `fadeTransition`s to `MatchScene` with the resulting `roomId` and `playerTokens`.
11. Given `createMatchRoom()` or `createMatch()` fails, When Player clicks `StartMatchButton`, Then `MatchSettingsScene` reports the error and does not transition to `MatchScene`.
12. Given `MatchSettingsScene.create()` runs (fresh boot or re-entry from `MatchScene`'s Return to Settings), Then it unconditionally `fadeIn`s, completing the `fadeTransition` pair regardless of which Scene preceded it.
