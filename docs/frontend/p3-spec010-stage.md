---
title: "Phase 3.10: Match Settings Scene - Stage Subscene"
---

# Match Settings Scene - Stage Subscene"

## Context

Phase 3.9 introduced Match Settings Scene with `UnitSubscreen`. This spec renders the remaing `StageSubscreen` to complate the flow of Match Settings, allowing Player to configure and start a match.

## Goal

- Render `StageSubscreen` in `MatchSettingsScene` to allow the Player to configure the match.
- Enter `MatchScene` via `MatchSettingsScene`.
- Start the app from `MatchSettingsScene`.

## Non-Goal

- Polished animations or tweens.
- VS COM or Online multi-player mode of `MatchSettingsScene`.

## Scene Entry

TBC.

In `MatchScene`, when `VictoryCutScene` is shown, Player can click `returnMatchSettingsButton` to transition to `MatchSettingsScene`. It should bring `gameCfg` to `MatchSettingsScene`, representing `MatchSettingsScene` remember the last match's settings.

## Shared Components

### BackButton

The functionality of the shared `BackButton` is similar to the one in `MatchScene`, but withi different rendering. It will be used in various scenes other that `MatchScene`.

It's a **96x96px** rounded squares filled with HEX `0x4c4c4c`. In the center render a ASCII `⮐` but **flipped horizontally** representing it's turning back. The font size is **36px** and font color is HEX `0xffffff`.

### Transitions

We generally use `200ms dim -> 200ms undim` as fading out then fading in transition effect for all scene to scene or subscene to subscene.

> Note: Agent should tell if we could define a name `200ms dim -> 200ms undim` which we generally use for scene transition so that we don't have to type this long keyword.

## Data Fetching

### Catalog

When `MatchSettingsScene` loads up, call `getCatalog()` to fetch available `archetypes` and `stagePresets`. Both of them are essential data sets for the game setup. If either one of them returns an empty list, report the error and do not allow the game to proceed. Since they are constants unless a redeployment. It's safe to store in `MatchSettingsScene`.

## MatchSettings Scene

`MatchSettingsScene` consists of 3 subscene:

- `UnitSubscene` for Player 1
- `UnitSubscene` for Player 2
- `StageSubscene`

Overall, the scene should leave **48px** margin for all 4 sides.

There could be more/fewer subscenes in future, esp for future we have other modes like VS COM or Online, so don't hardcode it as 3 only.

`MatchSettingsScene` divides into 3 sections:

- The top **108px** is the `HeaderSection`.
  - It should contain a `BackButton`, then 48px of spacer, then renders the title of the current subscene.
- The bottom **108px** is the `NavSection` which renders the Next/Confirm buttons.
- The middle main sections are for the subscenes.

> Note: Agent should check `subscene` and `section` is an appropriate term in Phaser and give a better name if needed.

### Store the Latest `gameCfg`

`MatchSettingsScene` keeps the latest `gameCfg` as a private scene-instance field. The main duty of the scene is to assist Player to construct the correct `gameCfg` and pass to backend to create a match.

It could carry `gameCfg` from the last match (see: [Scene Entry](#scene-entry)). But if there isn't, below is the default values of `gameCfg` for initialization:

```json
{
  stagePreset: 'Plain',
  p1Teams: ['King'],
  p2Teams: ['King'],
  maxTurns: 60,
  allowResetTurn: true,
}
```

## Unit Subscene

The title to be rendered in the `HeaderSection` is `[P{X}] Unit Selection`, where X is 1 or 2 as this is for both players. `P1` and `P2` are each wrapped in a rounded rectangle filled with that team's `TEAM_COLORS` (`constants.ts`) entry (team 1's color behind `P1`, team 2's color behind `P2`) — the same color used elsewhere for that team (e.g. `MatchSummary`).

In the main section, divide it verically into 2 sections, `FormationPanel` and `ArchetypesPanel`.

In the footer section, render a **144Wx96Hpx** rounded rectangle button (`NextButton`), using `PANEL_BUTTON_*` style in `constants.ts`. The button should contains `NEXT →` in the center. Align with `PanelButton` in `MatchScene`.

### Visual Effect of Formation Panel

`FormationPanel` covers **25%** of the left of `UnitSubscene`, which shows the team formation that the Player selects.

- Render a header `Formation` on the top, center aligned, in font size **12px**,  font color `0xffffff`.
- Render 5 **96x96px** rounded squares as a column, named `UnitSlot`, filled with team's `TEAM_COLORS` (`constants.ts`), with **12px** spacing.
  - A number is assigned and render in each `UnitSlot` From the top to the bottom. Render the number at the top left corner, in font size **8px**, font color `0xffffff`.
  - The number order from the top to the bottom is **4 -> 2 -> 1 -> 3 -> 5**, representing the order of `gameCfg.p{X}teams` to be passed to create the match. Note that as user interface, we use 1-base representation, but in coding we should always use 0-base representation.
  - Render the `order number` at the top left corner, leaving 4px each side, in font size **8px**,  font color `0xffffff`. 
    - Refer to `@engine/presets.go` `stagePresetsRegistry()` for this special order.
  - The middle `UnitSlot` **1** is for `King` only. Render a **96x96px** sprite of King in the team's `TEAM_COLORS` (`constants.ts`). Refactor `renderUnits()` and `drawArchetypeIcon()` in `@web/src/rendering/boardRenderer.ts` so that they could be reusable. Since `Unit's` sprite overlaps `order number`, the `Unit's` sprite should have a lower depth then `order number`.
  - [User Interaction and Visual Effect of Unit Subscene](#user-interaction-and-visual-effect-of-unit-subscene) will explain more on how to trigger the "put on the units and put down the units". When the `Unit` is put on the the selected `UnitSlot`, that `UnitSlot` should render the `Unit` like how `King` is rendered in `UnitSlot 1`.
- If `MatchSettingsScene.gameCfg` contains `p{X}teams`, render it accordingly.

### Visual Effect of Archetypes Panel

`ArchetypesPanel` covers the remaining **75%** of the `UnitSubscene`, which shows all `archetypes` obtained previously. Each `archetype` should convert to `Unit` represented by a `UnitCard`.

`ArchetypesPanel` lists all `UnitCards`. Each card should have **12px** spacing. At the moment there should be 4 `UnitCards` a row. Note that in future there will be more than 16 archetypes available, so `ArchetypesPanel` should be veritically scrollable.

#### Unit Card

`UnitCard` is a **144Wx192Hpx** rounded rectangle container, with **12px** padding on 4 side.

- All the elements should be center-aligned.
- All the text is in font size **12px**,  font color `0xffffff`.
- In `UnitCard`, render a **120x120px** rounded square container. Refactor `renderUnits()` and `drawArchetypeIcon()` in `@web/src/rendering/boardRenderer.ts` so that they could be reusable.

> Convention: `render*` owns creating and placing a GameObject in the scene graph; `draw*` paints shapes into a `Graphics` object the caller already created and passed in. `drawArchetypeIcon` already fits this — reuse it by having `UnitCard` pass in its own `Graphics`. Apply this rule to any new decoration helpers added by this spec.
- The remaining space in `UnitCard` should render the preset stats of an `Archetype`, using `archetype.name`, `archetype.speed`, `archeType.bombMaxRange` and `archeType.skills`.
- `archetype.speed`, `archeType.bombMaxRange` should be combined in 1 line. Use 👟 and 💣 in front of the values. Leave extra **12px** in between `archetype.speed` and 💣.
- At this moment, `archeType.skills` always contain at most 1 skill. Render `-` for `None`.

Below is the sample `UnitCard` of Witch for Team 1:

```text
+------------+
| +--------+ |
| | Witch  | |
| | sprite | |
| | (Blue) | |
| +--------+ |
|   Witch    |
| 👟 1  💣 3 |
|     -      |
+------------+
```

> Note: Current font `Roboto` may not render glyph correctly. Agent should explore a font if necessary, but **DO NOT** put much effort as all glyphs will be replaced by better representation sprite anyway.

### User Interaction and Visual Effect of Unit Subscene

- All `UnitCards` in `ArchetypesPanel` have click handlers.
  - If a `UnitCard` is clicked, that `Unit` should be put on the **lowest available** `UnitSlot` in `FormationPanel`.
    - e.g., If in `FormationPanel` `UnitSlot` 1 is `King`, `UnitSlots` 2 and 5 are `NO_UNIT`, `UnitSlots` 3 and 4 is `Fighter`, when I click `Witch` in `ArchetypePanel`, `Witch` should be put on `UnitSlot 2`.
  - If all `UserSlots` are full, `UnitCards'` click handlers should do nothing.
- All `UnitSlots` expcet the middle one (`King`) have click handlers.
  - Since `King` is always in the middle and can't be removed, there is no click handler for `King` in `UnitSlot`.
  - If the `UnitSlots` doesn't contain a unit, its click handler should do nothing.
  - Otherwise, the `Unit` in the clicked `UnitSlot` should be removed, meaning, to free a slot for user to select again from `FormationPanel`.
  All the put on/off should update `MatchSettingsScene.gameCfg.p{X}Teams` immediately.
- Under this interaction setup, it's possible to have partial teams in non-continuous order. Player can put on `Units` to all 5 `UnitSlots`, then click `UnitSlots` 2 and 4, causing the team only occupies `UnitSlots` 1, 3, and 5.
- `NextButton` should react on `FormationPanel's` data.
  - It should be disabled (with `DISABLED_BUTTON_*` style in `constants.ts`) if only `King` is in `FormationPanel`.
  - Button handler should be available when there are **2-5** units in `FormationPanel`.
    - Collect the `p{x}Teams` in string, e.g., If `UnitSlots` 1-5 are `King`, `Fighter`, <Empty>, `Witch`, `Witch` for Team 1, `p1Teams = ['King', 'Fighter', 'NO_UNIT', 'Witch', 'Witch']`. Ref `@engine/game.go` `NoUnit`.
    - If current subscene is `UnitScene` 1, transitions to `UnitScene` 1.
    - If current subscene is `UnitScene` 2, transitions to `StageSubscene`.
- `BackButton`
  - Since `TitleScene is not ready, `BackButton should have no effect if current subscene is `UnitScene` 1.
  - If current subscene is `UnitScene` 2, transitions to `UnitScene` 1.

## Stage Subscene

The title to be rendered in the `HeaderSection` is `Stage Selection`.

In the footer section, render a **144Wx96Hpx** rounded rectangle button (`StartMatchButton`), using `PANEL_BUTTON_*` style in `constants.ts`. The button should contains `Start Match` in the center. Align the style of  `NextButton` in `UnitSubscene`.

## Misc Updates other than Match Setting Scene

This spec also does cosmetic refinement in `MatchScene`:

- Instead of rendering `bomb` with **24×24px** circle, use **24px** `💣`.
  - **Countdown** should still be rendere on top of it.
- Instead of rendering `burn` with 2 triangles in `unitDamagedEvent` and `softBlockDestroyedEvent`, use **42px** `🔥`.

> Note:
> - Current font `Roboto` may not render glyph correctly. Agent should explore a font if necessary, but **DO NOT** put much effort as all glyphs will be replaced by better representation sprite anyway.
> - Though it's out of scope, but the reasons to include here:
>   1. Glyphs are closely related to `UnitCard` in this spec.
>   2. Small change to avoid overhead.
---

## Acceptance Criteria

1. Given … When … Then …
2. Given … When … Then …
