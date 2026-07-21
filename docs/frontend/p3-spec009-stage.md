---
title: "Phase 3.9: Match Settings Scene"
---

# Match Settings Scene

## Context

Phase 3.6 concludes a match and returns to Match Settings Scene, which is a blank scene. This spec renders all necessary settings in UI, allowing Player to configure and start a match.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- Render `MatchSettingsScene` to allow the Player to configure the match - Unit Selection only.

## Non-Goal

- Start the app from `MatchSettingsScene` - stick with `DevBootScene` -> `MatchScene` until `p3-spec010-stage.md`.
- Enter `MatchScene` via `MatchSettingsScene`.
- Polished animations or tweens.
- VS COM or Online multi-player mode of `MatchSettingsScene`.
- Strict mode in Unit Selection.

## Scene Entry

In `MatchScene`, when `VictoryCutscene` is shown, Player can click `returnMatchSettingsButton` to transition to `MatchSettingsScene`. It should bring `gameCfg` to `MatchSettingsScene`, so `MatchSettingsScene` remembers the last match's settings. (`MatchScene` already holds the match's `gameCfg` as a scene field, so it has the value to forward.)

## Data Fetching

### Catalog

When `MatchSettingsScene` loads up, call `getCatalog()` to fetch available `archetypes` and `stagePresets`. Both are essential data sets for the game setup. If either returns an empty list, report the error and do not allow the game to proceed. Since they are constants unless a redeployment, it's safe to store them in `MatchSettingsScene`.

Because this fetch is async and fired on scene load, guard against scene shutdown: if `MatchSettingsScene` is no longer active when the promise resolves, discard the result instead of building GameObjects. (This is a fresh instance of the async-in-`create()` pattern already logged as `p3-spec002-match-log.md` issue #1 — guarded here proactively so we don't ship a second unguarded copy.)

## MatchSettings Scene

`MatchSettingsScene` consists of 3 `Pages`:

- `UnitPage` for Player 1
- `UnitPage` for Player 2
- `StagePage`

Overall, the scene should leave **48px** margin for all 4 sides.

There could be more/fewer `Pages` in future, esp. once we add modes like VS COM or Online, so don't hardcode it as 3 only.

`MatchSettingsScene` divides into 3 `regions`:

- The top **108px** is the `HeaderRegion`.
  - It contains a `BackButton`, then 48px of spacer, then the title of the current `Page`.
- The bottom **108px** is the `NavRegion`, which renders the Next/Start Match buttons.
- The middle region is the body that holds the active `Page`.

### Store the Latest `gameCfg`

`MatchSettingsScene` keeps the latest `gameCfg` as a private scene-instance field. The main duty of the scene is to assist Player to construct the correct `gameCfg` and pass to backend to create a match.

It could carry `gameCfg` from the last match (see: [Scene Entry](#scene-entry)). But if there isn't, below is the default values of `gameCfg` for initialization:

```json
{
  "stagePreset": "Plain",
  "p1Teams": ["King"],
  "p2Teams": ["King"],
  "maxTurns": 60,
  "allowResetTurn": true
}
```

## Unit Page

The title to be rendered in the `HeaderRegion` is `[P{X}] Unit Selection`, where X is 1 or 2 as this is for both players. `P1` and `P2` are each a `TeamBadge` filled with that team's `TeamColor` (team 1's color behind `P1`, team 2's color behind `P2`) — the same color used elsewhere for that team (e.g. `MatchSummary`).

In the body region, divide it vertically into two `Panels`, `FormationPanel` and `ArchetypesPanel`.

In the `NavRegion`, render a **144Wx96Hpx** pill-shaped button (`NextButton`) — a `PillButton` whose size is overridden to `pill(144, 96)`, keeping the `PANEL_BUTTON_*` fill/border colors (so the size overrides `PANEL_BUTTON_WIDTH`/`HEIGHT` of 46×32). The button contains `NEXT →` centered.

### Visual Effect of Formation Panel

`FormationPanel` covers **25%** of the left of `UnitPage`, which shows the team formation that the Player selects.

- Render a header `Formation` on the top, center aligned, in font size **24px**, font color `0xffffff`.
- Render 5 **96x96px** rounded squares as a column, named `UnitSlot`, filled with team's `TeamColor`, with **12px** spacing.
  - Each `UnitSlot` shows an `order number` at its top-left corner (4px inset, font size **24px**, color `0xffffff`), assigned top-to-bottom in the order **4 → 2 → 1 → 3 → 5**. This is the 1-based display order of `gameCfg.p{X}Teams` (code stays 0-based). Refer to `@engine/presets.go` `stagePresetsRegistry()` `P1StartingPositions` for this order.
  - The middle `UnitSlot` (order number **1**) is `King`-only. Render a **96x96px** King sprite in the team's `TeamColor`. The unit sprite must render *below* the `order number` (lower depth), since they overlap.
    - **Reuse note:** `renderUnits()` in `@web/src/rendering/boardRenderer.ts` is board-coupled (derives position from `unit.position`, hardcodes `UNIT_SIZE` 32px, board depth, and board click handlers), so it can't be reused as-is for a **96px** off-board sprite. Extract the per-unit drawing into a helper taking an explicit `(cx, cy, size)` and team color (e.g. `drawUnitSprite`) that both the board and this `Page` call. `drawArchetypeIcon()` already fits the `draw*` convention (see [Unit Card](#unit-card)); pass it the slot's own `Graphics` and scale the icon to `size` rather than the fixed `OCCUPANT_ICON_RADIUS` (10px), which would look tiny at 96px.
  - [User Interaction](#user-interaction-and-visual-effect-of-unit-page) explains how "put on / take off units" is triggered. When a `Unit` is put on a `UnitSlot`, that slot renders the `Unit` the same way `King` is rendered in the middle slot.
- If `MatchSettingsScene.gameCfg` contains `p{X}Teams`, render it accordingly.

### Visual Effect of Archetypes Panel

`ArchetypesPanel` covers the remaining **75%** of the `UnitPage`, which shows all `archetypes` obtained previously. Each `archetype` should convert to `Unit` represented by a `UnitCard`.

`ArchetypesPanel` lists all `UnitCards`. Each card should have **12px** spacing. At the moment there should be 4 `UnitCards` a row. Note that in future there will be more than 16 archetypes available, so `ArchetypesPanel` should be vertically scrollable.

#### Unit Card

`UnitCard` is a **180Wx240Hpx** rounded rectangle container, with **12px** padding on 4 side.

- All the elements should be center-aligned.
- All the text is in font size **24px**, font color `0xffffff`.
- In `UnitCard`, render a **96x96px** rounded square holding the unit sprite, drawn via the same reusable `drawUnitSprite`/`drawArchetypeIcon` helpers described under [Formation Panel](#visual-effect-of-formation-panel) (here `size` = 96px).

> Convention: `render*` owns creating and placing a GameObject in the scene graph; `draw*` paints shapes into a `Graphics` object the caller already created and passed in. `drawArchetypeIcon` already fits this — reuse it by having `UnitCard` pass in its own `Graphics`. Apply this rule to any new decoration helpers added by this spec.

- The remaining space in `UnitCard` should render the preset stats of an `Archetype`, using `archetype.name`, `archetype.speed`, `archetype.bombMaxRange` and `archetype.skills`.
- `archetype.speed`, `archetype.bombMaxRange` should be combined in 1 line. Use 👟 and 💣 in front of the values. Leave extra **12px** in between `archetype.speed` and 💣.
- At this moment, `archetype.skills` always contain at most 1 skill. Render `-` if `archetype.skills` is empty.

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

> Note: These glyphs are placeholders; a formal sprite replaces all of them eventually, so keep effort minimal. Research: 👟 `U+1F45F` and 💣 `U+1F4A3` are Unicode 6.0 emoji, covered by every major platform emoji font (Apple Color Emoji, Segoe UI Emoji, Noto Color Emoji) — browsers fall back to it even on canvas, so they render (no tofu ▯; only the per-OS look varies). **Backup plan:** if manual testing shows a glyph rendering incorrectly, swap it for a plain-text form (e.g. `Spd 1  Bomb 3`).

### User Interaction and Visual Effect of Unit Page

- All `UnitCards` in `ArchetypesPanel` have click handlers.
  - If a `UnitCard` is clicked, that `Unit` should be put on the **lowest available** `UnitSlot` in `FormationPanel`.
    - e.g., If in `FormationPanel` `UnitSlot` 1 is `King`, `UnitSlots` 2 and 5 are `NO_UNIT`, `UnitSlots` 3 and 4 is `Fighter`, when I click `Witch` in `ArchetypesPanel`, `Witch` should be put on `UnitSlot 2`.
  - If all `UnitSlots` are full, `UnitCards'` click handlers should do nothing.
- All `UnitSlots` except the middle one (`King`) have click handlers.
  - Since `King` is always in the middle and can't be removed, there is no click handler for `King` in `UnitSlot`.
  - If the `UnitSlot` doesn't contain a unit, its click handler should do nothing.
  - Otherwise, the `Unit` in the clicked `UnitSlot` should be removed, freeing that slot for the user to select again from `ArchetypesPanel`.
    All put-on/take-off should update `MatchSettingsScene.gameCfg.p{X}Teams` immediately.
- Under this interaction setup, it's possible to have partial teams in non-continuous order. Player can put on `Units` to all 5 `UnitSlots`, then click `UnitSlots` 2 and 4, causing the team only occupies `UnitSlots` 1, 3, and 5.
- `NextButton` should react on `FormationPanel`'s data.
  - It should be disabled (rendered as `DisabledButton`, filled with `DISABLED_BUTTON_COLOR` in `constants.ts`) if only `King` is in `FormationPanel`.
  - Button handler should be available when there are **2-5** units in `FormationPanel`.
    - Collect the `p{X}Teams` in string, e.g., If `UnitSlots` 1-5 are `King`, `Fighter`, <Empty>, `Witch`, `Witch` for Team 1, `p1Teams = ['King', 'Fighter', 'NO_UNIT', 'Witch', 'Witch']`. Ref `@engine/game.go` `NoUnit`.
    - If current `Page` is `UnitPage` 1, `fadeTransition` to `UnitPage` 2.
    - If current `Page` is `UnitPage` 2, `fadeTransition` to `StagePage`.
- `BackButton`
  - Since `TitleScene` is not ready, `BackButton` has no effect when the current `Page` is `UnitPage` 1.
  - If current `Page` is `UnitPage` 2, `fadeTransition` to `UnitPage` 1.

## Stage Page

The title to be rendered in the `HeaderRegion` is `Stage Selection`.

In the `NavRegion`, render a **144Wx96Hpx** pill-shaped button (`StartMatchButton`) styled identically to `NextButton` (same `pill(144, 96)` size and `PANEL_BUTTON_*` colors). The button contains `Start Match` centered.

The body region will be specified in `p3-spec010-stage.md`. This spec only needs a stub `Page`.

## Misc Updates other than Match Setting Scene

This spec also does cosmetic refinement in `MatchScene`:

- Instead of rendering `bomb` with **24×24px** circle, use **24px** `💣`.
  - **Countdown** should still be rendered on top of it.
- Instead of rendering `burn` with 2 triangles in `unitDamagedEvent` and `softBlockDestroyedEvent`, use **42px** `🔥`.

> Note:
>
> - Placeholder glyphs; formal sprites replace them eventually, so keep effort minimal. Research: 💣 `U+1F4A3` and 🔥 `U+1F525` are Unicode 6.0 emoji, so OS-emoji fallback guarantees they render (no tofu ▯; per-OS look only). **Backup plan:** if manual testing shows incorrect rendering, revert to the prior shapes (24×24 circle / 2-triangle burn) or a text form.
> - Out of scope but bundled here because: (1) glyphs relate closely to `UnitCard`; (2) small change, avoids overhead.

---

## Acceptance Criteria

1. Given `UnitPage` is currently active for Player 1, `ArchetypesPanel` should render all `archetypes` from `archetypesRegistry()` in backend as `UnitCard` and fill in Blue.
2. Given `UnitPage` is currently active for Player 2, `ArchetypesPanel` should render all `archetypes` from `archetypesRegistry()` in backend as `UnitCard` and fill in Red.
3. Given `getCatalog()` returns an empty `archetypes` or `stagePresets` list, When `MatchSettingsScene` loads, Then it should report the error and not allow the game to proceed.
4. Given `MatchScene` transitions to `MatchSettingsScene`, `UnitPage` should render the Unit Selections using `gameCfg` brought from `MatchScene`.
5. Given any `UnitPage`, the middle `UnitSlot` should always hold `King`, have no click handler, and cannot be removed.
6. Given `FormationPanel` contains a free slot, When Player clicks a `UnitCard`, Then that `Unit` should be placed on the **lowest available** `UnitSlot`.
7. Given `FormationPanel` doesn't contain a free slot, When Player clicks any `UnitCard`, Then `FormationPanel` should do nothing.
8. Given `FormationPanel` has a `UnitSlot` other than `King` occupied, When Player clicks that `non-King UnitSlot`, Then the clicked `UnitSlot` should be freed.
9. Given Player puts on / takes off `Units` (possibly leaving non-contiguous gaps), Then `gameCfg.p{X}Teams` should immediately reflect the slots, with `NO_UNIT` in each empty non-King slot (e.g. `['King', 'Fighter', 'NO_UNIT', 'Witch', 'Witch']`).
10. Given only `King` is in `FormationPanel`, Then `NextButton` should render as a `DisabledButton`.
11. Given `FormationPanel` has at least 2 `UnitSlots` occupied including `King`, When Player clicks `NextButton`, Then the active `Page` should `fadeTransition` to the next `Page`.
12. Given `FormationPanel` has less than 2 `UnitSlots` occupied including `King`, When Player clicks `NextButton`, Then `NextButton` should do nothing.
13. Given `UnitPage` is currently active for Player 1, When Player clicks `BackButton`, Then `BackButton` should do nothing.
14. Given `UnitPage` is currently active for Player 2, When Player clicks `BackButton`, Then `UnitPage` should transition to `UnitPage` for Player 1.
15. Given `StagePage` is currently active, When Player clicks `BackButton`, Then `StagePage` should transition to `UnitPage` for Player 2.

## Log

Implementation issues found during the build (non spec gaps) are tracked in [p3-spec009-stage-log.md](./p3-spec009-stage-log.md).
