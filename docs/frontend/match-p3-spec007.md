---
title: "Phase 3.7 Surrender, Reset, and MatchSummaryPanel"
---

# Surrender

As of Phase 3.6, the Players have to play the whole game to go back to `MatchSettingsScene`. Phase 3.7 provides a way to end the match prematurely. This phase also allows Players to reset the sandbox (GameState.WorkingState) back to original (GameState.TrueState) for rollback, and groups all 3 `TurnLifeCycleButtons` into `MatchSummaryPanel` instead of leaving on `MatchScene` randomly.

## Goal

- Add `SurrenderButton` to restart a match.
- Add `ResetTurnButton` to reset the `WorkingState`.
- Render `MatchSummaryPanel` to keep all 3 `TurnLifeCycleButtons`

## Non-Goal

- HUD.

## Scene Entry

No change from spec006.

## TurnLifeCycle Buttons

There are 3 Turn Life-cycle operations in the game, `ResolveTurn`, `ResetTurn` and `Surrender`. Unlike `TurnCommand` which manipulate the `WorkingState`, Turn Life-cycle operations manipulate the whole turn data.

**Interaction lock contract (applies project-wide):** Any action that triggers a server call — including `ResolveTurnButton`, `ResetTurnButton`, `SurrenderButton` here, and `ConfirmDialog`'s `yesButton` for `moveButton`/`placeBombButton` (`TurnCommandPanel`, see `match-p3-spec003.md`) — must disable all user interactions the instant the call is triggered, and only re-enable them once the server has responded (success or error). Re-rendering/animation is a parallel concern and must not gate when interactions re-enable.

## MatchSummary Panel

`MatchSummaryPanel` is a panel rendered on top of `MatchScene`.

## MatchSummary Button

`MatchScene` renders a **48x48px** rounded square at the top right hand corner, which should mirror the position of `TurnPanel`, leaving 48px space from the top and right edges. Its depth is same as `TurnCommandPanel`. The button contains a menu symbol `≡` in font color `0xffffff` and font size **48px**

When the Player clicks `MatchSummaryButton`, `MatchSummaryPanel` will be rendered as the below section. 

### Visual Effect of MatchSummary Panel

The `MatchSummaryPanel` fades in in **200ms**, stays on `MatchScene` until the Player closes it and fades out in **200ms**. **All user interactions disabled except the buttons in `MatchSummaryPanel` until this panel is closed.**

- A dim background layer (semi-transparent scrim, consistent with `ConfirmDialog`'s dim background) covering **100% width, 100% height**.
- The panel's content area is centered on screen, **640Wx640Hpx**, with **48px** margin from the screen edges on all sides.
  - Font color and size are `0xffffff` and **48px**. The components are center-aligned within their own column.
  - The top **15%** of the panel is for displaying `gameCfg.stagePreset` and `gameCfg.maxTurns`. Render these in a 2-column style.
  - The next **35%** of the panel is for displaying the match data. Render these in a 3-column style.
    - Living Units can be counted by `units` with `HP > 0` per Team.
    - Available Bombs can be counted by `unit.maxBombCount - unit.bombUsed` for each `unit` with `HP > 0` per Team.
  - The bottom half of this panel is for 3 `TurnLifeCycleButtons` and  `MatchSummaryPanelBackButton`
    - Move `ResolveTurnButton` originally in `MatchScene` to `MatchSummaryPanel`.
    - Render `ResetTurnButton`, `SurrenderButton` and `MatchSummaryPanelBackButton` below `ResolveTurnButton`. Each button should leave **12px** gap at the bottom.
    - All `Yes` handlers in `ConfirmDialog` triggered by 3 `TurnLifeCycleButtons` should start with closing `MatchSummaryPanel`, followed by their corresponding actions.

Sample representation for the transparent panel:
  ```text
  +-------------------------------------+
  |                                     |
  |      Stage                MAP01     |
  |    Max Turns               30       |
  |                                     |
  |                                     |
  |   P1                          P2    |
  |    5       Living Units        3    |
  |    2      Available Bombs      4    |
  |                                     |
  |                                     | 
  |         [ResolveTurnButton]         |
  |          [ResetTurnButton]          |
  |          [SurrenderButton]          |
  |     [MatchSummaryPanelBackButton]   |
  |                                     |
  +-------------------------------------+
```

## Surrender Button

`SurrenderButton` is one the Game Lifecycle Command buttons, which falls in the same category of `ResolveTurnButton`. Therefore, the rendering spec (fill, border, font) should stay consistent. [Follow how `ResolveTurnButton` is rendered, and how Player interacts with `ResolveTurnButton`](match-p3-spec004.md#resolveturn-button).

The only 3 differences are:

- `SurrenderButton` contains text `Surrender`.
- `ConfirmDialog` shows `Confirm to surrender?`
- After the Player chooses `Yes`, move on to [Surrender handling](#surrender-visual-effect-and-interaction)

### Click Handler of Surrender Button

- Interactions lock on click, per the [Interaction lock contract](#turnlifecycle-buttons), and stay locked until `surrender()` responds.
- Call surrender(). `matchEndedEvent` should be returned from the backend.
- Render `VictoryCutscene` just as when match is concluded during `resolveTurn`.

## Reset Button

> Note: `ResetTurn` is a **user-initiated** turn rollback only. It is **not** the client's
> error-recovery path — a rejected/failed command resyncs via `getMatchState()` per
> `match-p3-spec008.md` (Render-Path Contract, caller (c)), which must not route through Reset (that
> would discard the turn's other planned actions). Reset's masked re-render is caller (b) of that
> same contract.

Same visual effect as `SurrenderButton`, except:

- `Surrender` -> `Reset this turn`
- `Confirm to surrender?` -> `All the actions made during this turn will be reset. Confirm?`
- If `gameCfg.allowResetTurn = false`, `ResetTurnButton` is disabled, change all the color to `0x4c4c4c`.

### Click Handler and Visual Effect of Reset Button

After clicking this button, a series of actions will be executed:

- Interactions lock on click, per the [Interaction lock contract](#turnlifecycle-buttons), and stay locked through `resetTurn()`'s response.
- In parallel, dim the whole canvas in **200ms**, just like fading out, to mask the re-render.
- While dimming the screen, call `resetTurn()` to notify the backend to `ResetTurn()`.
- If the response is not **HTTP 200**, log the error in `ErrorPanel`.
- If the response is **HTTP 200**, re-fetch and re-render from `getMatchState()`. After that, go back to [Game Loop #5.4](match-p3-spec005.md#game-loop).
- After the re-rendering completes, undim the whole canvas in **200ms**, just like fading in. Interactions re-enable once `resetTurn()` has responded — this is independent of when the dim/undim/re-render visuals finish.
> Note: ResetTurn() rollback to the state **after** Sudden Death hazard being injected. There is no need to re-render Sudden Death related animations.

## MatchSummaryPanelBack Button

The visual effect is as same as the 3 `TurnLifeCycleButton`. Unlike those buttons, there is no `ConfirmDialog` handling.

`MatchSummaryPanelBackButton` contains text `Back`. If the Player clicks this button, it closes `MatchSummaryPanel` as the way stated in [above](#visual-effect-of-matchsummary-panel), so that Player can resume the gameplay.

---

## Acceptance Criteria

1. Given `MatchSummaryButton`, When Player clicks it, Then `MatchSummaryPanel` should appear with `gameCfg`, current match state, 4 buttons.
2. Given Player 1 clicks `SurrenderButton`, `VictoryCutscene` should be shown with Player 2 as the winner.
3. Given Player 2 clicks `SurrenderButton`, `VictoryCutscene` should be shown with Player 1 as the winner.
4. Given Player 1 clicks `ResetTurnButton`, `MatchScene` should revert to the state the Player 1 started, with SuddenDeath already injected if it's in Sudden Death state.
5. Given Player 2 clicks `ResetTurnButton`, `MatchScene` should revert to the state the Player 2 started, with SuddenDeath already injected if it's in Sudden Death state.
6. Given Player clicks `MatchSummaryPanelBackButton`, `MatchSummaryPanel` should be closed and Player should be able to continue to navigate the occupants.
