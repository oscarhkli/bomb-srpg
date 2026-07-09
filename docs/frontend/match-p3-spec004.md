---
title: "Phase 3.4: Render Resolve Turn"
---

# Render Resolve Turn

## Context

This spec adds turn system command `ResolveTurn` control that trigger the server calculation and render the outcomes based to the `gameEvents` received.

## Goal

- `MatchScene` renders `TurnPanel` showing the current turn.
- `MatchScene` renders `ResolveTurnButton`, allowing Player to end the turn.
- Player can end the turn and see the effects after the backend calculation, e.g., `unit` died, `bomb` detonated, etc..

## Non-Goal

- `OptionPanel` Showing the summary and the system command buttons - Phase 3.5+.
- `ResetTurnButton` - Phase 3.5+.
- Polished animations (mild effect is fine).
- HUD / status panel.

## Layout

### Store the Latest `gameCfg`

`MatchScene` keeps the latest `gameCfg` as a private scene-instance field (same pattern already used for `roomId`/`playerTokens`, e.g. `private gameCfg!: GameCfg;`)

### Bomb / SoftBlock Graphics Lookup

`MatchScene` adds `bombGraphicsById: Map<number, Phaser.GameObjects.Graphics>` and `softBlockGraphicsById: Map<number, Phaser.GameObjects.Graphics>` private fields, mirroring the existing `unitGraphicsById` pattern. `renderBombs`/`renderSoftBlocks` populate these maps by `bomb.id`/`softBlock.id` as they create each `Graphics` object.

This is required so `gameEvent` handlers below (`bombCountdownUpdated`, `bombExploded`, `softBlockDestroyed`) can look up and mutate/destroy the specific rendered object for a given ID, instead of scanning `boardObjects`.

### Turn Panel

`TurnPanel` is rendered as **96Wx48Hpx** rounded square panel at the top left hand corner of the `MatchScene`, leaving 48px space from the top and left edges. It's depth is same as `TurnCommandPanel`.

`TurnPanel` display 4 elements: A text `Turn`, `gameState.turn`, followed by a text `/`, then `gameCfg.maxTurns`. The rough sample display is shown below:

```text
+-----------+
| Turn      |
|    2 / 30 |
+-----------+
```

- `TurnPanel` should leave 8px padding on each side.
- The top `Header` section should fill depends on the `gameState.activeTeam`. Use `TEAM_COLORS` in `constants.ts`.
- The bottom `Value` section should be transparent. The whole text is right aligned, but to avoid the text shifting due to the number, 3 text should be split into independent elements and on the exact location. The maximum value of both numeric values is **99**.
- Font family is `GAME_FONT_FAMILY`, `0xeeeeee`. If `gameState.turn` > `gameCfg.maxTurns`, render `gameState.turn` in HEX `0xff0000` indicating the match is now in sudden death.

**Remarks:** The dimensions and positions are temporary values and will be adjust during the implemenatation.

### ResolveTurn Button

`MatchScene` should render `ResolveButton` **48px** below its top edge. This is a **640Wx72** center-aligned text `End this turn`. Font family is `GAME_FONT_FAMILY`, font color is `PANEL_BUTTON_BORDER_COLOR`.

A click handler should be added to `ResolveButton`, opening `ConfirmDialog` with text `Confirm to end this turn?`. If Player clicks `No`, `ConfirmDialog` will be closed. If Player clicks `Yes`, it will trigger `ResolveTurn`.

## ResolveTurn and the Subsequent Visual Effects

After the Player confirms to end the turn, frontend should call the backend via `resolveTurn()`. Just as the other APIs, it request `roomId` and correct `playerToken` to operate. Also, print the errorMessage if there is.

If the API is correctly executed, the backend will do to all the heavy lift, and return a series of `gameEvents`. 

### BombCountDownUpdatedEvent

`bombCountDownUpdatedEvent` should at least contain `bombId` and `countdown`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `bomb` with `bombCountDownUpdatedEvent.bombId` exists in `gameState.bombs`.

For each `bombCountDownUpdatedEvent`: 

1. Re-render the number shown in the `bomb` using `bombCountDownUpdatedEvent.countdown`. If countdown is **0**, render a `!` with HEX `#0xff0000` instead.

### BombExplodedEvent

`bombExplodededEvent` should at least contain `bombId` and `affectedPositions`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `bomb` with `bombExplodededEvent.bombId` exists in `gameState.bombs`.
- `grid` contains all the coordinates in `affected`, i.e., no out-of-bound problem.

This event renders a cardinal-ray blasting effect of a `bomb`. For each `bombExplodededEvents`:

1. Look up the `bomb's` position.
2. Sort `affectedPositions` by **Manhattan distance**.
3. Remove the `bomb` image from the `grid`.
4. The animation should start from `bomb's` position, extending its blast length to the outermost in 4 directions simultaneously, at speed of **100ms per tile**.
    - The blast should be rendered in 3-layer gradient color. The outermost starts with HEX `0xf58e27`, then `0xf5ee27`, to the innermost `0xfcfabb`. Opacity is **60%**.
5. If `tile.occupantType` is not `OccupantNone`, render a fire shape on top of the blast and occupant, representing the occupant is burning. Opacity is *70%**.

> Note: Work with agent on how to best describe the blast rendering.

The blasting effect should last for **5s** from the blast center.

### UnitDamagedEvent

Skip it.

### UnitDiedEvent

`unitDiedEvent` should at least contain `unitId`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `unit` with `unitDiedEvent.unitId` exists in `gameState.units`.

For each `unitDiedEvents`:

1. Look up the `unit's` position.
2. Render a cross shape in HEX `0xff0000` on top of the `unit`.
3. **3s** later, remove the `unit` of the cross.

### SoftBlockDestroyedEvent

`softBlockDestroyedEvent` should at least contain `softBlockId`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `softBlock` with `softBlockDestroyedEvent.softBlockId` exists in `gameState.softBlocks`.

For each `softBlockDestroyedEvents`:

1. Look up the `softBlock's` position.
2. Render a cross shape in HEX `0xff0000` on top of the `softBlock`.
3. **3s** later, remove the `softBlock` of the cross.

### Rendering Sequence

This is the trickiest part. `Bombs` explodes with chain-reaction, meaning they could be exploded even if `countdown > 0`. Although backend has already sorted `gameEvents` chronologically, it doesn't explicitly state which `bombs` are exploded due to chain-reaction. An `Occupant` shouldn't be affected until the blast animation reaches its own `tile`.

```text
bombCountDownUpdatedEvent-1; bombId-001, countdown-0
bombCountDownUpdatedEvent-2; bombId-002, countdown-0
bombCountDownUpdatedEvent-3; bombId-003, countdown-2
bombExplodedEvent-4; bombId-001
bombExplodedEvent-5; bombId-003
bombExplodedEvent-6; bombId-003
unitDamagedEvent-7; unitId-007
unitDiedEvent-8; unitId-007
softBlockDestroyedEvent-9; softBlockId-009
unitDamagedEvent-10; unitId-010
unitDiedEvent-11; unitId-010
```

In the above example of `gameEvents` returned.

1. `bombCountDownUpdatedEvent-1`, `bombCountDownUpdatedEvent-2`, `bombCountDownUpdatedEvent-3` should be rendered concurrently first as they are the 1st group of `gameEvents`.
2. `bombExplodedEvent-4` and `bombExplodedEvent-5` should be rendered concurrently as their `countdown` reach **0**.
3. `bombExplodedEvent-6` should be rendered only when the blast reaches `bombId-003`.
4. `unitDamagedEvent-7`, `softBlockDestroyedEvent-9` and `unitDamagedEvent-10` should be rendered only when the blast reaches the corresponding `Occupant`.
5. `unitDiedEvent-8` and ``unitDiedEvent-11` should be rendered after each of `unitDamagedEvent-7` and `unitDamagedEvent-10` handled. 

> Note: Should discuss with Agent to see if backend need enhancement, e.g., adding `bombChainReactedEvent`.


## Refresh Final Sanity Check

Same operation as stated in [match-p3-spec004.md](match-p3-spec004.md#refresh-final-sanity-check).

---

## Acceptance Criteria

1. Given `TurnPanel` is rendered, when `gameState.activeTeam` changes, then the header fill color updates to match `TEAM_COLORS[activeTeam]`.
2. Given `gameState.turn` > `gameCfg.maxTurns`, when `TurnPanel` renders, then the turn number renders in `0xff0000`.
3. Given a `bombCountdownUpdated` event with `countdown > 0`, when rendered, then the bomb's displayed number updates to the new countdown.
4. Given a `bombCountdownUpdated` event with `countdown === 0`, when rendered, then the bomb shows a red `!` instead of a number.
5. Given a `bombExploded` event, when rendered, then affected tiles show the blast effect and the bomb sprite is removed from the grid.
6. Given a `bombExploded` event whose `affectedPositions` includes an occupied tile, when rendered, then a fire shape renders on top of that occupant.
7. Given a `unitDied` event, when rendered, then a red cross appears over the `unit`; after 3s the `unit` is removed.
8. Given a `softBlockDestroyed` event, when rendered, then a red cross appears over the `softBlock`; after 3s the `softBlock` is removed.
9. Given a gameEvent missing a required field or referencing a non-existent bombId/unitId/softBlockId, when encountered, then an error message is shown and rendering halts per "game unable to proceed."