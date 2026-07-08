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

### Turn Panel

`TurnPanel` is rendered as **96Wx48Hpx** rounded square panel at the top left hand corner of the `MatchScene`, leaving 48px space from the top and left edges. It should be at the 3rd top z-index.

> Note: The 1st top will be `OptionPanel`, which will be added in Phase 3.5+. The 2nd top is `ConfirmDialog`. Adjust it if current application doesn't act like that.

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

> Note: 
> - The dimensions and positions are temporary values and will be adjust during the implemenatation. 
> - Sudden Death rendering will be included in Phase 3.5+. Changing color is just a representation.

### ResolveTurn Button

`MatchScene` should render `ResolveButton` **48px** below its top edge. This is a **640Wx72** center-aligned text `End this turn`. Font family is `GAME_FRONT_FAMILY`, font color is `PANEL_BUTTON_BORDER_COLOR`.

A click handler should be added to `ResolveButton`, opening `ConfirmDialog` with text `Confirm to end this turn?`. If Player clicks `No`, `ConfirmDialog` will be closed. If Player clicks `Yes`, it will trigger `ResolveTurn`.

## ResolveTurn and the Subsequent Visual Effects

After the Player confirms to end the turn, frontend should call the backend via `resolveTurn()`. Just as the other APIs, it request `roomId` and correct `playerToken` to operate. Also, print the errorMessage if there is.

If the API is correctly executed, the backend will do to all the heavy lift, and return a series of `gameEvents`. Render them in the order of the result array.

### BombCountDownUpdatedEvent

### BombExplodedEvent

### UnitDamagedEvent

### UnitDieEvent

### SoftBlockDestroyedEvent

---

## Acceptance Criteria

1. TBC
