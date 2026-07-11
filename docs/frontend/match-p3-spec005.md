---
title: "Phase 3.5: Turn Lifecycle Wiring (startTurn)"
---

# Turn Lifecycle Wiring (startTurn)

## Context

`MatchScene` never calls `startTurn()` (`engine/api.ts`) today. Per `AGENTS.md`'s Turn Lifecycle rules, the client must explicitly call `POST /match/start-turn` for every turn, including Turn 1, to evaluate sudden-death / hazard injection. Phase 3.3 (Move/PlaceBomb) and Phase 3.4 (Resolve Turn) works around token initialization by re-deriving `initToken()` once per `TurnCommandPanel` open instead of once per turn boundary.

## Goal

- Wire `startTurn()` into the turn flow at the correct point(s).
- Move `initToken()` to fire once per `startTurn()` resolution instead of once per panel-open or resolving turn, since `activeTeam` is stable between turns.

## Non-Goal

- Polished animations or tweens (easing curves, squash/stretch, particle effects, etc.) — a bare linear-motion tween for `suddenDeath` is in scope, see [Visual Effect for `unitMovedEvent`](#visual-effect-for-unitmovedevent).
- HUD / status panel.

## Scene Entry

No change from spec001.

## Start Turn

There are 4 steps in `startTurn`:

1. Store the contract `token` for this turn. See [below section](#player-token-handling).
2. Call backend via `startTurn()`.
3. Render `suddenDeath` bomb drop if the criteria satisfy.
4. Render a cutscene to indicate it's P1/P2's turn.

During the whole `startTurn` timeframe, **all** interactions/handlers in `MatchScene` should be **disabled**.

Details are shown in the following sections.

## Player Token Handling

Currently, `initToken()` is called in when unit is clicked and during `resolveTurn()`. However, the active token does not change until the turn is changed. From now on, `initToken()` should be called in `startTurn()` instead.

Additionally, `MatchScene` no longer needs to `console.log` the `roomId` and `playerTokens` during the initialization. This addresses `match-p3-spec002-log.md` #12.

## Sudden Death

In this game, `suddenDeath` is triggered when `startTurnResponse.inSuddenDeath = true`, which is from the response of `startTurn()`.

In `StartTurn()`, backend calls `injectSuddenDeathHazards()`. As of Phase 3.5, 0-2 `bombPlacedEvents` will be returned. It's possible not to have any `bombPlacedEvents` received from the backend. As always, the frontend should trust what the backend provides.

### Visual Effect of Sudden Death

- Check if `startTurnResponse.inSuddenDeath = true`.
- If not, end as the game hasn't reach sudden death yet.
- If conditions are fulfilled,
  - Render an red warning cutscene (regardless the number of `bombPlacedEvents` received):
    - It lasts for **3s**.
    - Add a full canvas overlay in `TURN_PANEL_SUDDEN_DEATH_COLOR` in `constants.ts`.
    - This overlay has repeating opacity change from **0%** to **90%** to **0%** in 500ms.
  - **2s** after the red warning cutscene starts rendering, for each `bombPlacedEvents`:
    - Render `bombPlacedEvent` similar to the way in `match-p3-spec003.md`, with a tween:
    - We want to present the scenario as "drop the `bomb` from the sky"
    - The `x` position should be the same as `bombPlacedEvent.position.x`.
    - The `y` position should be as high as fully above the visibld screen.
    - Then the `bomb` should slide to the designated `tile` as stated in `bombPlacedEvent.position.y` in a straight line in **2s**.

> Note: Timing value will be updated during the implementation

## Visual Effect of Start Turn Cutscene

As a simplified version, `MatchScene` renders a **100% width, 144px height** rectangle banner. Use `TEAM_COLORS` in `constants.ts`. `Player {X}'s Turn` is rendered in the center of the banner. Font color is `0xffffff`. Font size is **48px**

This Cutscene fades in in **200ms**, stays on `MatchScene` for **2sec** and fades out in **200ms**.

## Game Loop

The section states the whole game loop as of Phase 3.5. `MatchScene` may have to adjust accordingly.

> Question to agent:
>
> 1. Since we have no plan to change tileType in the mid-game, should we render the grid in the beginning instead re-rendering everytime in `renderBoard()`? The current flow does redundant rendering.
> 2. Since we trust what the backend provides, the frontend mostly only do the rendering and player's interaction, is current sanity check really necessary, or YAGNI?
> 3. If sanity check isn't necessary, do we really need to re-render every time when we refresh `gameState`?

1. `MatchScene` is launched by `LoungeScene` after a successful `createMatch()`. **All user interactions disabled.**
2. `roomId` and `playerTokens` are stored in `MatchScene`.
3. Render the environment, i.e., `grid`.
4. `getMatchState()` to render the `occupants` and `TurnPanel`.
5. Start the Game Loop:
   1. `getMatchState()` and update `TurnPanel`. **All user interactions disabled.**
   2. `StartTurn` and possible drop the `bombs` due to `suddenDeath`.
   3. **All user interactions enabled.**
   4. (Optional) Player's interaction loop:
      1. If `move`, **All user interactions disabled.**. Then `move` the unit according to `unitMovedEvent`.
      2. If `placeBomb`. **All user interactions disabled.**. Then render a new `bomb` according to `bombPlacedEvent`.
      3. **All user interactions enabled.**
   5. Player clicks `resolveTurn`. **All user interactions disabled.**
   6. Render all `gameEvents` returned from backend's `ResolveTurn`.
   7. (To be done in `match-p3-spec006.md`) Check `victoryResult` and break the Game Loop in the match has concluded. Otherwise, go back to **5.1**.

---

## Acceptance Criteria

1. Given `Start Turn Cutscene` is rendered, when `gameState.activeTeam` changes, then the fill color should match `TEAM_COLORS[activeTeam]`.
2. Given `startTurnResponse.inSuddenDeath = true`, when `startTurn` renders, then red warning cutscene should be shown, and `bombs` should drop from the sky according to the number of `bombPlacedEvents` received.
3. `roomId` and `playerTokens` shouldn't be seen in `console.log` again.
