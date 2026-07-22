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

## StagePage

The main region of `StagePage` consists of 2 panels, `StagesPanel` and `StageDetailPanel`, sitting in 2 columns. `StagePage` receives `stagePresets` (from `MatchSettingsScene`'s catalog) and the shared `gameCfg` reference, mirroring how `UnitPage` already receives `archetypes`/`gameCfg`.

### Visual Effect of Stages Panel

`StagesPanel` covers 50% of `StagePage`. Each side should have **12px** padding.

`StagesPanel` lists all `StageCards` in a row, center aligned. Each card should have **12px** spacing. Note that in future there will be more than 8 stagePresets available, so `StagesPanel` should be horizontally swipable. Since there is not enough StagePresets to scroll at the moment, the swipe effect is a **non-goal** unless it's very simple to do.

#### Stage Card

`StageCard` is a **160Wx160Hpx** rounded rectangle container, with **12px** padding on 4 side. [User Interaction](#user-interaction-and-visual-effect-of-stage-page) will describe a **Select** behavior. When a `StageCard` is selected, add a border of **4px** in HEX `0xdc9e23`.

- For simplicity, `stagePreset.name` should be rendered at the center of the card, with font size **48px**.
- If `MatchSettingsScene.gameCfg` contains `stagePreset`, select it according. Otherwise, select the first `StageCard`.

### Visual Effect of StageDetail Panel

`StagesDetailPanel` covers 50% of `StagePage`. Draw a rectangle panel of size 80% of `StagesDetailPanel`, namely `InnerPanel`. Each side of `InnerPanel` should have **12px** padding.

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
- `{description}`: Left aligned with 1 extra line reserved for line wrapping.
- `{width} x {height}`: Center aligned. `x` should be at the dead center.
- `{maxTurns}`: This is a selection component named `MaxTurnsSelector`, with a `❰` and `❱` wrapped.
  - 2 arrows symbol should be at a fixed position, i.e. **24px** at the nearest edge. The font color is HEX `0xdc9e23`.

### User Interaction and Visual Effect of Stage Page

- All `StageCards` in `StagesPanel` have click handlers.
  - If a `StageCard` is clicked, the `StageCard` is selected. The other `StageCard` should be unselected.
  - `StagesDetailPanel` should get the infomation from `stagePresets` and replace all the information inside.
  - `maxTurns` is a cyclic selection component (`MaxTurnsSelector`). The available selections are: `💣`, `15`, `20`, `30`, `45`, `60`. Note that `💣` means `0` in behind.
    - Selecting *any* `StageCard` — including re-selecting one whose `maxTurns` was previously customized — always resets `MaxTurnsSelector` to `stagePreset.maxTurns` (that stage's recommended value), discarding any prior customization for that stage. Unlike the other components, it never restores the value from `MatchSettingsScene.gameCfg`.
    - Add a click handler for `❰` - clicking this will shift the selection left, e.g., `20 -> 15`. Note that it's cyclic, i.e., `💣 -> 60`.
    - Add a click handler for `❱` - clicking this will shift the selection right, e.g., `15 -> 20`. Note that it's cyclic, i.e., `60 -> 💣`.
    - If the currently selected value is `stagePreset.maxTurns`, render a `🌟` with font size **8px** (placeholder size — refine during implementation) next to the value, meaning it's recommended max turns. Note that it should be a fix position - never move any elements to accommodate it.
  - Any user interaction in `StagePage` should update `MatchSettingsScene.gameCfg.stagePreset` and `MatchSettingsScene.gameCfg.maxTurns` immediately.
- `StartMatchButton`:
  - Since 1 and only 1 `StageCards` is always selected, validation isn't necessary in `StagePage`, i.e., `StartMatchButton` should always enabled.
  - If Player clicks `StartMatchButton`, follow how `DevBootScene` calls `create()`:
    - Call `createMatchRoom()` to get `roomId`.
    - Use `roomId` to `initRoom()`.
    - Use `roomId` and `MatchSettingsScene.gameCfg` to call `createMatch()`.
    - `fadeTransition` to `MatchScene` with `roomId` and `playerTokens`.

---

## Acceptance Criteria

1. Given both Player 1 and Player 2 configure their `Units` in `UnitPage`, and they select the `Stage` in `StagePage`, When Players click `StartMatchButton`, the game creates the `Room` and `Match` and transitions to `MatchScene`.
