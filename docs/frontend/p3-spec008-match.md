---
title: "Phase 3.8: Surrender, Reset, and MatchSummaryPanel"
---

# Surrender

As of Phase 3.6, the Players have to play the whole game to go back to `MatchSettingsScene`. Phase 3.8 provides a way to end the match prematurely. This phase also allows Players to reset the sandbox (GameState.WorkingState) back to original (GameState.TrueState) for rollback, and groups all 3 `TurnLifeCycleButtons` into `MatchSummaryPanel` instead of leaving on `MatchScene` randomly.

## Goal

- Add `SurrenderButton` to restart a match.
- Add `ResetTurnButton` to reset the `WorkingState`.
- Render `MatchSummaryPanel` to keep all 3 `TurnLifeCycleButtons`.

## Non-Goal

- HUD.

## Scene Entry

No change from spec006.

## TurnLifeCycle Buttons

There are 3 Turn Life-cycle operations in the game, `ResolveTurn`, `ResetTurn` and `Surrender`. Unlike `TurnCommand` which manipulate the `WorkingState`, Turn Life-cycle operations manipulate the whole turn data.

**Interaction lock contract (applies project-wide):** Any action that triggers a server call — including `ResolveTurnButton`, `ResetTurnButton`, `SurrenderButton`, `MatchSummaryButton` here, and `ConfirmDialog`'s `yesButton` for `moveButton`/`placeBombButton` (`TurnCommandPanel`, see `p3-spec003-match.md`) — must disable all user interactions the instant the call is triggered, and only re-enable them once the server has responded (success or error). Re-rendering/animation is a parallel concern and must not gate when interactions re-enable.

"All user interactions" explicitly includes `TurnCommandPanel`'s own buttons (Move/Bomb/Back) and board unit-click handling, not just the 4 buttons named above — a unit click or a `TurnCommandPanel` button press during an in-flight `moveButton`/`placeBombButton` submission must be a no-op until that submission's response arrives. (This closes the gap tracked in `p3-spec003-match-log.md` issue #4, which was deferred here pending spec008.)

## MatchSummary Panel

`MatchSummaryPanel` is a panel rendered on top of `MatchScene`.

## MatchSummary Button

`MatchScene` renders a **48x48px** rounded square at the top right hand corner, which should mirror the position of `TurnPanel`, leaving 48px space from the top and right edges. Its depth is same as `TurnCommandPanel`. The button contains a menu symbol `≡` in font color `0xffffff` and font size **48px**

`MatchSummaryPanel`'s own depth sits above `TurnPanel`/`TurnCommandPanel` (so it blocks board interaction underneath) but below `ConfirmDialog` (so the `Yes`/`No` prompt from a `TurnLifeCycleButton` click still renders on top of the panel).

When the Player clicks `MatchSummaryButton`, `MatchSummaryPanel` will be rendered as shown below.

### Visual Effect of MatchSummary Panel

The `MatchSummaryPanel` fades in in **200ms**, stays on `MatchScene` until the Player closes it and fades out in **200ms**. **All user interactions disabled except the buttons in `MatchSummaryPanel` until this panel is closed.**

- A dim background layer (semi-transparent scrim, consistent with `ConfirmDialog`'s dim background) covering **100% width, 100% height**.
- The panel's content area is centered on screen, **720Wx640Hpx**.
  - Font color is `0xffffff` and size is **36px**. Row spacing must give each line comfortable breathing room rather than packing rows tightly — the components are center-aligned within their own column.
  - The top **15%** of the panel is for displaying `gameCfg.stagePreset` and `gameCfg.maxTurns`. Render these in a 2-column style.
  - The next **35%** of the panel is for displaying the match data. Render these in a 3-column style.
    - Living Units can be counted by `units` with `HP > 0` per Team.
    - Available Bombs can be counted by `unit.maxBombCount - unit.bombUsed` for each `unit` with `HP > 0` per Team.
    - `P1` and `P2` are each wrapped in a rounded rectangle (**96x48px**) filled with that team's `TEAM_COLORS` entry (team 1's color behind `P1`, team 2's color behind `P2`) — the same color used elsewhere for that team (e.g. `TurnPanel`'s header).
  - The bottom half of this panel is for 3 `TurnLifeCycleButtons` and `MatchSummaryPanelBackButton`.
    - These 4 buttons are **44px** tall (down from `ResolveTurnButton`'s original **72px**, `p3-spec004-match.md#resolveturn-button`), width unchanged. Font size is unchanged (not scaled down with the button). `VictoryCutscene`'s `rematchButton`/`returnMatchSettingsButton` (`p3-spec006-match.md`) are resized to this same **44px** height for visual consistency — width and font unchanged there too.
    - The 4-button block is bottom-aligned within the panel — flush against the content box's bottom edge, preserving a **12px** gap after the last button.
    - Move `ResolveTurnButton` originally in `MatchScene` to `MatchSummaryPanel`, keeping its existing click behavior unchanged (`p3-spec004-match.md#resolveturn-button`) — including force-closing any open `TurnCommandPanel` action before showing its `ConfirmDialog`.
    - Render `ResetTurnButton`, `SurrenderButton` and `MatchSummaryPanelBackButton` below `ResolveTurnButton`, each leaving **12px** gap from the one above it.
    - All `Yes` handlers in `ConfirmDialog` triggered by 3 `TurnLifeCycleButtons` should start with closing `MatchSummaryPanel`, followed by their corresponding actions.

Sample representation for the transparent panel:
  ```text
  +-------------------------------------+
  |                                     |
  |      Stage                MAP03     |
  |                                     |
  |    Max Turns                 6      |
  |                                     |
  |  [P1]                       [P2]    |
  |                                     |
  |    5        Living Units      5     |
  |                                     |
  |   12       Available Bombs   12     |
  |                                     |
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

`SurrenderButton` is one the Game Lifecycle Command buttons, which falls in the same category of `ResolveTurnButton`. Therefore, the rendering spec (fill, border, font) should stay consistent. [Follow how `ResolveTurnButton` is rendered, and how Player interacts with `ResolveTurnButton`](p3-spec004-match.md#resolveturn-button).

The only 3 differences are:

- `SurrenderButton` contains text `Surrender`.
- `ConfirmDialog` shows `Confirm to surrender?`
- After the Player chooses `Yes`, move on to [Surrender handling](#click-handler-of-surrender-button)

### Click Handler of Surrender Button

- Interactions lock on click, per the [Interaction lock contract](#turnlifecycle-buttons), and stay locked until `surrender()` responds.
- Call `surrender()` with the currently **active team**'s `teamId` (`gameState.activeTeam`) — under today's pass-and-play mode, both Players share one client, so "who clicked" is derived from whose turn it currently is, not a separate per-Player session.
  > Note: This is a pass-and-play placeholder. Once online multiplayer exists, this must instead resolve to the client's own registered team (from its `playerToken`/session), regardless of which team is currently active — a future spec's concern, not this one's.
- `matchEndedEvent` should be returned from the backend.
- Render `VictoryCutscene` just as when match is concluded during `resolveTurn`.

## Reset Button

> Note: `ResetTurn` is a **user-initiated** turn rollback only. It is **not** the client's
> error-recovery path — a rejected/failed command resyncs via `getMatchState()` per
> `p3-spec007-match.md` (Render-Path Contract, caller (c)), which must not route through Reset (that
> would discard the turn's other planned actions). Reset's masked re-render is caller (b) of that
> same contract.

Same visual effect as `SurrenderButton`, except:

- `Surrender` -> `Reset this turn`
- `Confirm to surrender?` -> `All turn actions will reset. Confirm?`
- If `gameCfg.allowResetTurn = false`, `ResetTurnButton` is disabled, change all the color to `DISABLED_BUTTON_COLOR`.

### Click Handler and Visual Effect of Reset Button

After clicking this button, a series of actions will be executed:

- Interactions lock on click, per the [Interaction lock contract](#turnlifecycle-buttons), and stay locked through `resetTurn()`'s response.
- In parallel, dim the whole canvas in **200ms**, just like fading out, to mask the re-render.
- While dimming the screen, call `resetTurn()` to notify the backend to `ResetTurn()`.
- If the response is not **HTTP 2xx**, log the error in `ErrorPanel`.
- If the response is **HTTP 2xx**, re-fetch via `getMatchState()` and rebuild the **occupant layer only** — the occupant-only wholesale swap defined by `p3-spec007-match.md`'s Render-Path Contract (caller (b)). The terrain layer is not rebuilt. After that, go back to [Game Loop #5.4](p3-spec005-match.md#game-loop).
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
7. Given a `TurnCommandPanel` button or unit click occurs while a turn-command or turn-lifecycle server call is in flight, then the click is a no-op until the response arrives.
8. Given `MatchSummaryButton` is clicked while interactions are locked, then the click is a no-op.
9. Given `ResetTurnButton` succeeds, then only occupant graphics are rebuilt from the fresh state — the terrain layer is untouched.

## Log

Implementation issues found during the build (non spec gaps) are tracked in [p3-spec008-match-log.md](./p3-spec008-match-log.md).
