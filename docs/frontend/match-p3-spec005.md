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

- Polished animations or tweens (easing curves, squash/stretch, particle effects, etc.) — a bare linear-motion tween for `suddenDeath` is in scope, see [Visual Effect of Sudden Death](#visual-effect-of-sudden-death).
- HUD / status panel.

## Scene Entry

No change from spec001.

## Start Turn

There are 4 steps in `startTurn`:

1. Store the active team's `playerToken` for this turn. See [below section](#player-token-handling).
2. Call backend via `startTurn()`.
3. If `startTurnResponse.inSuddenDeath` is true, render the sudden-death sequence (the `SuddenDeathCutscene`, then bomb drop). See [below](#visual-effect-of-sudden-death).
4. Render the `TurnBanner` to indicate it's P1/P2's turn.

Steps 3 and 4 run **sequentially**: on a sudden-death turn the `SuddenDeathCutscene` (step 3) plays to completion first, then the `TurnBanner` (step 4); on a normal turn step 3 is skipped and only the `TurnBanner` plays.

During the whole `startTurn` timeframe, **all** interactions/handlers in `MatchScene` should be **disabled**.

Details are shown in the following sections.

## Player Token Handling

Currently, `initToken()` is called in when unit is clicked and during `resolveTurn()`. However, the active token does not change until the turn is changed. From now on, `initToken()` should be called in `startTurn()` instead.

The active team's token is `playerTokens[activeTeam - 1]`. `initToken(playerTokens[activeTeam - 1])` must fire **before** the `startTurn()` API call — `startTurn()` sends `authHeaders()` and returns 401 without a token. Since `activeTeam` comes from the preceding `getMatchState()`, the order is `getMatchState()` → `initToken()` → `startTurn()`.

Additionally, `MatchScene` no longer needs to `console.log` the `roomId` and `playerTokens` during the initialization. This addresses `match-p3-spec002-log.md` #12. Removing that log also requires updating the coupled `MatchScene.test.ts` assertion (`'logs roomId and playerTokens on create'`, which asserts on literal token values) so the suite stays green.

## Sudden Death

In this game, `suddenDeath` is triggered when `startTurnResponse.inSuddenDeath` is true, which is from the response of `startTurn()`.

In `StartTurn()`, backend calls `injectSuddenDeathHazards()`. As of Phase 3.5, 0-2 `bombPlaced`-typed entries in `startTurnResponse.gameEvents` (referred to below as `bombPlacedEvents`) will be returned — from `startTurn()`, all `gameEvents` are `bombPlaced`. It's possible not to have any `bombPlacedEvents` received from the backend. As always, the frontend should trust what the backend provides.

Because `injectSuddenDeathHazards()` has already committed these bombs server-side by the time `startTurn()`'s response returns, `MatchScene` must also refresh its tracked `GameState` via `getMatchState()` when `inSuddenDeath` is true — **before** rendering the cutscene/bomb-drop — so `gameState.bombs` includes them. This isn't just for rendering: a later `resolveTurn()` may report `bombCountdownUpdated`/`bombExploded` events referencing these bomb ids, and the client-side event validation checks those ids against the tracked `gameState.bombs`. Skipping this refresh leaves `gameState.bombs` stale and causes resolveTurn event validation to fail with an "unknown bombId" error.

### Visual Effect of Sudden Death

- Check if `startTurnResponse.inSuddenDeath` is true.
- If not, end as the game hasn't reach sudden death yet.
- If conditions are fulfilled,
  - Render the **`SuddenDeathCutscene`** (regardless of how many `bombPlaced` events arrive):
    - A full-canvas `Rectangle` pinned to the camera (`scrollFactor 0`), filled with `TURN_PANEL_SUDDEN_DEATH_COLOR` (`constants.ts`), drawn above all board content.
    - Its alpha **pulses** via a yoyo, repeating `Tween`: `0 → 0.9` then back to `0` (one full pulse = **500ms**), looping for the **3s** `SuddenDeathCutscene` duration, then the rectangle is destroyed.
  - **2s** after the `SuddenDeathCutscene` starts rendering, for each `bombPlaced` event in `startTurnResponse.gameEvents` (present it as the bomb dropping from the sky):
    - Resolve the target tile's world-space center via `tileCenter(bombPlacedEvent.position)` (the same helper used in `match-p3-spec003.md` / `match-p3-spec004.md`) — `bombPlacedEvent.position` is a grid coordinate, not pixels.
    - Create the bomb graphic as in `match-p3-spec003.md`, but positioned **off-screen**: same world `x` as the target tile's center, with `y` set high enough that the bomb starts **fully above the visible board**.
    - `Tween` its `y` straight down to the target tile's world-space center `y` over **2s**. On complete it rests on the tile like a normally-placed bomb.

> Note: Timing value will be updated during the implementation

## TurnBanner

As a simplified version, `MatchScene` renders the **`TurnBanner`**: a **100% width, 144px height** rectangle banner. Use `TEAM_COLORS` in `constants.ts`. `Player {X}'s Turn` is rendered in the center of the banner. Font color is `0xffffff`. Font size is **48px**.

The `TurnBanner` fades in in **200ms**, stays on `MatchScene` for **2sec** and fades out in **200ms**.

## Game Loop

The section states the whole game loop as of Phase 3.5. `MatchScene` may have to adjust accordingly.

> Render-path performance (render grid once, drop redundant re-render / sanity checks) is out of scope — tracked in `match-p3-spec008.md`.

1. `MatchScene` is launched by `LoungeScene` after a successful `createMatch()`. **All user interactions disabled.**
2. `roomId` and `playerTokens` are stored in `MatchScene`.
3. Render the environment, i.e., `grid`.
4. `getMatchState()` to render the `occupants` and `TurnPanel` — the initial full-board render.
5. Start the Game Loop:
   1. **All user interactions disabled.**
   2. `getMatchState()`, `initToken()` and update `TurnPanel`. This is a per-turn **refresh** — whether it re-renders the board or only refreshes the interaction maps is the spec011 question noted above.
   3. `startTurn()`: render the `TurnBanner`, and — if `inSuddenDeath` — refresh `gameState` via `getMatchState()` (see [Sudden Death](#sudden-death)) before rendering the `SuddenDeathCutscene` + dropped `bombs`.
   4. **All user interactions enabled.**
   5. (Optional) Player's interaction loop:
   6. If `move`, **All user interactions disabled.**. Then `move` the unit according to `unitMovedEvent`.
   7. If `placeBomb`. **All user interactions disabled.**. Then render a new `bomb` according to `bombPlacedEvent`.
   8. **All user interactions enabled.**
   9. Player clicks `resolveTurn`. **All user interactions disabled.**
   10. Render all `gameEvents` returned from backend's `ResolveTurn`.
   11. (To be done in `match-p3-spec006.md`) Check `victoryResult` and break the Game Loop in the match has concluded. Otherwise, go back to **5.1**.

---

## Acceptance Criteria

1. Given the `TurnBanner` is rendered, when `gameState.activeTeam` changes, then the fill color should match `TEAM_COLORS[activeTeam]`.
2. Given `startTurnResponse.inSuddenDeath` is true, when `startTurn` renders, then the `SuddenDeathCutscene` should be shown, and `bombs` should drop from the sky according to the number of `bombPlaced`-typed entries in `startTurnResponse.gameEvents` (`bombPlacedEvents`) received.
3. `roomId` and `playerTokens` shouldn't be seen in `console.log` again.
4. Given a unit is moved in consecutive turns with no intervening grid change, when each turn's `unitMoved` renders, then the unit graphic ends centered on the tile reported by `gameState` for that turn.
5. Given `startTurn` is in progress (the `TurnBanner`/`SuddenDeathCutscene` or bomb-drop playing), when the player clicks a unit or tile, then no handler fires; interactions re-enable only after the sequence completes.
6. Given a turn begins, when `startTurn()` is called, then `initToken(playerTokens[activeTeam - 1])` has already fired for this turn (the request carries auth headers, no 401), and `initToken` is no longer invoked on unit-click or during `resolveTurn`.
7. Given any turn begins, when the `TurnBanner` renders, then a banner reading `Player {X}'s Turn` (X = `gameState.activeTeam`) is shown and then dismissed before interactions re-enable.
