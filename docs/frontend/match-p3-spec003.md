---
title: "Phase 3.3: Render Move and PlaceBomb"
---

# Render Move and PlaceBomb

## Context

Phase 3.1 adds a click handler on `occupants`. This spec adds the actionable interactions on `unit`: `Move` and `PlaceBomb`.

## Goal

- Player can command their own team's `unit` to move or place bomb, provided that it is legal to do so.
- Player can view the allowedTiles per `TurnCmdType`
- `MatchScene` renders `Unit` movement and new `Bombs` after receiving server's confirmation.

## Non-Goal

- Polished animations or tweens (easing curves, squash/stretch, particle effects, etc.) — a bare linear-motion tween for `Move` is in scope, see [Visual Effect for `unitMovedEvent`](#visual-effect-for-unitmovedevent).
- HUD / status panel.
- Player can click and view the information of `unit` and `bomb`.
- Warning message about data out-of-sync with backend and frontend.

## Scene Entry

No change from spec001.

## Notes on Coloring

All the color HEX below will be adjusted in future to align with pixel style palettes. Always define in `constants.ts` for better maintenance.

## User Interaction and Data Fetching

This spec consists of various user interactions. Some of them require data fetching. Some of them submit action to server and react based on the response.

### TurnCommandPanel

`TurnCommandPanel` is rendered as **192Wx144Hpx** rectangle when needed (to be explained later). It's transparent, with no border. This contains the actions available for Player so choose. To simplify in the early stage. There are 3 buttons in the panel, `moveButton`, `placeBombButton`, `backButton`, distributed in 2 rows as illustrated below:

```text
+-------------+
| Move   Bomb |
|        Back |
+-------------+
```

All 3 buttons are rendered as **46Wx32Hpx** pill-shape, filled with `0x583f0e` and opacity **20%**, with **2px** `0xdc9e23` border color opacity **100%**. Text color is also the same color and opacity as border. Font family is `GAME_FONT_FAMILY` (see [Font](#font) below). Buttons are spaced `PANEL_BUTTON_SPACING` (**12px**) apart, both horizontally (`Move`↔`Bomb`) and vertically (`Bomb`↔`Back`).

`TurnCommandPanel` is positioned at the `Grid`'s right edge (a fixed gutter to the right of the last column), bottom-aligned with the `Grid`'s bottom edge. It does not follow or reposition per clicked `unit`. Unlike `ConfirmDialog` (see [Confirm Dialog](#confirm-dialog)), the panel is anchored to the grid in world space — it scrolls with the camera rather than staying pinned to the screen, since its position is spatially meaningful relative to the board.

### Font

This spec introduces the game's first UI text, so it establishes the default font for the current phase: a single web font, loaded via a `<link>` tag in `index.html` (e.g. Google Fonts), referenced everywhere through one `constants.ts` value, `GAME_FONT_FAMILY` (e.g. `"'Roboto', sans-serif"` — the quoted family plus a generic fallback). `Roboto` is not mandated; any single web-safe font is acceptable as long as it's centralized behind this one constant so it can be swapped later without touching call sites.

### Store the Latest `gameState`

`MatchScene` keeps the latest `gameState` as a private scene-instance field (same pattern already used for `roomId`/`playerTokens`, e.g. `private gameState!: GameState;`), not a separate store or global — this is a single-scene game with no cross-scene sharing need yet. It is set once in `create()` from the initial `getMatchState()` response, and reassigned whenever a later section obtains a fresh `GameState` (see [Refresh Final Sanity Check](#refresh-final-sanity-check)).

### Initialize Player Token

`playerTokens` (received from `DevBootScene` and stored on `MatchScene`, see spec001) is a tuple `[team1Token, team2Token]`. Before `submitTurnCommand()` can be called, the frontend must call `initToken(playerTokens[gameState.activeTeam - 1])` from `engine/api.ts` — this is currently never invoked anywhere, which would make every `submitTurnCommand()` call throw. Call it once, at the moment `TurnCommandPanel` opens for a `unit` (i.e., right after the User Interaction check below passes). Since this spec never triggers `resolveTurn()`, `activeTeam` cannot change mid-flow, so calling it again on a later click of the same team is a harmless no-op.

> See `match-p3-spec010.md` (stub) for re-deriving this per `startTurn()` instead, once that lifecycle call is wired up.

### User Interaction

`TurnCommandPanel` can only be shown and enabled when all the below situations are satisfied:

1. A `unit` is clicked.
2. `unit.team` equals to `state.activeTeam`.

Violating either means the unit is read-only. It prints a `console.log()` just as the current phase.

Additionally:

- If `unit.hasMoved` is **true**, `moveButton` should be disabled and rendered with HEX `0x999999`.
- If `unit.hasUsedSkill` is **true**, `placeBombButton` should be disabled and rendered with HEX `0x999999`. 

> Non-goal note:  
> Since currently we're working on Phase 3, where the game is pass-and-play. Both players are in the same client browser tab. For multiplayer online mode, additional rules should be add for displaying and enabling `TurnCommandPanel`: The current client's team equals to `state.activeTeam`. 

### Move

If Player clicks `moveButton`, first check the cache described in [Performance consideration](#performance-consideration) for key `(unitId, "move")`. On a cache hit, skip the network call and render the cached `tiles` directly. On a miss, call backend via `getAllowedTiles()`, using the payload:

```json 
{
  unitId: ${unit.id},
  turnCmdType: "move"
}
```

If non-200 HTTP result returns, display the errorMessage like how we handle Network error.

If 200 HTTP result returns with `AllowedTilesResponse`, these are the coordinates of the `tiles` that `unit` can move. For all matched `tiles`:

- Add a highlight overlay: HEX `0x86c64f` at opacity **65%**, layered on top of the tile.
- Add a click handler: When clicked,
  - The border of the highlight overlay should change to `0xdaedca` with opacity **100%**.
  - Confirm if Player wants to execute move (Details in [Confirm Dialog](#confirm-dialog) below).
    - If no, rollback to displaying `AllowedTiles`
    - If yes, use the selected `Coordinates` to `submitTurnCommand`
      ```json
        {
          type: "move";
          unitId: ${unit.id};
          target: ${the selected Coordinate};
        }
      ```
    - `GameEvent` Handling and the follow-up action will be described in next section.

### Confirm Dialog

When Player clicks a highlighted tile (for `Move` or `PlaceBomb`), `ConfirmDialog` appears: a **160Wx100Hpx** rectangle, centered on screen, filled with `0x1a1a1a` at opacity **60%** (dims the scene behind it), containing a short prompt text ("Confirm?") and two pill-shape buttons, `yesButton`/`noButton`, styled identically to `TurnCommandPanel`'s buttons (see [TurnCommand Panel](#turncommand-panel)).

`ConfirmDialog` is owned and instantiated by `MatchScene` directly, not by `TurnCommandPanel` — it's a scene-wide modal, not a panel-specific concern. `TurnCommandPanel` triggers it via injected callbacks (`showConfirm`/`hideConfirm`/`isConfirmOpen`) rather than holding its own instance. Because it must stay centered on screen regardless of where the camera has scrolled to (e.g. after `centerCamera` centers the view on the grid), every element of `ConfirmDialog` is pinned to the camera viewport (Phaser's `setScrollFactor(0)`) rather than drawn in world space.

- Player clicks `yesButton` → proceed with `submitTurnCommand` as described above.
- Player clicks `noButton` → dismiss `ConfirmDialog` and rollback to displaying `allowedTiles` (selected tile's border reverts, click handlers restored).

### Place Bomb

The mechanism `placeBomb` is similar to `move`. To avoid repeating:

- `moveButton` -> `placeBombButton`
- `turnCmdType move` -> `turnCmdType placeBomb`
- move -> place a bomb
- `0xdaedca` -> `0xe69138`
- `0xdaedca` -> `0xf7dec3`

### Back

`TurnCommandPanel` should contains an `actionStack`, where Player's actions are recorded, and used for rollback when Player clicks the rollback button. Below scenario is an example:

```text
Player clicks a unit -> Scene shows TurnCommandPanel
Player clicks moveButton -> Scene shows allowedTiles layer in 0xdaedca 
Player clicks a tile -> Scene asks for confirmation
Player clicks "No" -> Scene shows allowedTiles layer in 0xdaedca
Player clicks backButton -> Scene hides allowedTiles layer
Player clicks placeBombButton -> Scene shows allowedTiles layer in 0xe69138
... (and it continues)
```

If Player clicks `backButton` and there is nothing to rollback, `TurnCommandPanel` should then be hidden.

While `ConfirmDialog` is open, `TurnCommandPanel`'s own buttons (including `backButton`) are not interactive — the dialog's `noButton` is the only rollback path out of that state, and it performs the same rollback `backButton` would perform at that stack depth (see [Confirm Dialog](#confirm-dialog)).

There are two distinct ways `TurnCommandPanel` closes — they must not be conflated:

- **Incremental (`backButton`)**: pops one level off `actionStack`, rolling back to the prior visual state. Only hides the panel once the stack is already empty.
- **Immediate (system-driven)**: any close that happens as a *side effect* rather than the player navigating back — after `submitTurnCommand()` resolves (success or failure, per [Follow-up Actions](#follow-up-actions-after-turncommand-submission)), or when the player interacts with a different unit, resets the turn (future phase), etc. — hides `TurnCommandPanel` **and clears `actionStack` entirely**, not a single pop. Without the full clear, clicking the same unit again would reopen the panel carrying stale stack entries from the action that just finished.

### Performance consideration

Resolved: cache `getAllowedTiles()` results, keyed per `(unitId, turnCmdType)`, held on `MatchScene` itself — not scoped to a single `TurnCommandPanel` session. Populate a key lazily the first time it's needed — i.e., on the first click of `moveButton`/`placeBombButton` for that `unit` (see [Move](#move)), not when the panel opens. A cache hit skips the network call entirely, including across switching between units (A → B → A reuses A's cached tiles).

Invalidation must be a full clear (every key, every unit), not a per-unit one — because a successful action can change reachability for units *other* than the one that acted (e.g. a new bomb now blocks a different unit's path). The cache is cleared whenever `WorkingState` actually mutates:

- `submitTurnCommand()` **succeeds** (per `engine/match.go`, `CommandMoveUnit`/`CommandPlaceBomb` return their error *before* mutating `WorkingState` on any validation failure, so a failed submission changes nothing and the cache stays valid).
- `resetTurn()` — discards `WorkingState` and re-clones `TrueState`.
- `resolveTurn()` — commits the sandbox to `TrueState` and advances the turn; see `match-p3-spec0044.md` (Parked Draft) for where this gets wired into the frontend.

Neither `resetTurn()` nor `resolveTurn()` has frontend wiring yet (this spec doesn't add it) — flagging this now so whichever spec implements them doesn't miss clearing this cache. `resetTurn()` has no tracked spec yet.

## Follow-up Actions after TurnCommand Submission

`TurnCommandPanel` should be closed no matter if `submitTurnCommand()` is successful or not — this is the "Immediate (system-driven)" close case described in [Back](#back), so `actionStack` must be cleared entirely here too, not just popped once.

If non-200 HTTP result returns, display the errorMessage like how we handle Network error. The [Refresh Final Sanity Check](#refresh-final-sanity-check) still runs after this, unconditionally, on both the success and failure paths — a failed submission indicates a problem on the frontend side (the backend is the source of truth), so re-fetching and re-rendering from `getMatchState()` happens regardless of `submitTurnCommand()`'s outcome.

The below states the follow-up action when 200 HTTP result returns:

`submitTurnCommand()` returns `gameEvents` (already implemented in `engine/api.ts`/`types/api.ts`, verified against `server/http_handlers.go` as of this spec).

Currently we focus on `move` and `placeBomb` action. If there is any `gameEvent` with `type` other than `unitMoved` or `bombPlaced`, log it with `console.log()`.

### Visual Effect for `unitMovedEvent`

`unitMovedEvent` should contain at least `unitId`, `from` and `to`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `unitMovedEvent.unitId` equals to `unitId` of current actor.
- `grid[unitMovedEvent.from.y][unitMovedEvent.from.x].occupantType` equals `'OccupantUnit'` and its `occupantId` matches `unitMovedEvent.unitId`, i.e., the chosen `unit` is still in the `from` position.
- `grid[unitMovedEvent.to.y][unitMovedEvent.to.x].occupantType` equals `'OccupantNone'`, i.e., can be occupied by the chosen `unit`.
- `unitId` exists in `gameState.units` and `position` matches with `unitMovedEvent.from`.

Movement renders as a mild straight-line slide: tween the `unit` sprite's position from `from` to `to` in a single linear motion (Phaser `Tweens`/`TweenManager`, `ease: 'Linear'`, duration `UNIT_MOVE_TWEEN_DURATION` = 500ms). This is plain positional interpolation, not the "polished animations" this spec's Non-Goal excludes — no easing curves, squash/stretch, or particle effects. A **Manhattan-path** tween (multi-segment, following the actual pathfinding route instead of a straight line) remains a future polish candidate, not a decision this spec needs to make.

### Visual Effect for `bombPlacedEvent`

`bombPlacedEvent` should at least contain `bombId`, `unitId`, `position` and `countdown`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `bombPlacedEvent.unitId` equals to `unitId` of the chosen `unit`.
- `grid[bombPlacedEvent.position.y][bombPlacedEvent.position.x].occupantType` equals `'OccupantNone'`, i.e., can be occupied by the new `bomb`.

> Note: `engine/match.go`'s `IsLandingLegal()` enforces one occupant per tile — a `unit` and a `bomb` (or any two occupants) can never share a tile in this engine. The `topOnly`-click ambiguity flagged in `match-p3-spec002-log.md` (issue #4) therefore cannot occur for `unit`/`bomb` clicks; that issue can be marked resolved/moot when next updating that log.

## Refresh Final Sanity Check

After all `gameEvents` are handled (or after a failed `submitTurnCommand()` — see [Follow-up Actions](#follow-up-actions-after-turncommand-submission)), call `getMatchState()` once for sanity check. Compare the freshly-fetched `grid` against a client-side prediction — the pre-action `grid` with the validated event's mutation(s) applied (e.g. the moved `unit`'s new position, the newly placed `bomb`) — not against the raw pre-action `gameState`. Comparing against the pre-action state directly would always differ after any successful action and produce a false-positive mismatch every turn, so that comparison must not be used. If the fetched `grid` differs from the prediction, flag it in errorMessage, then re-render the `tiles`, `units` and `bombs` using new `gameState` as the same way as Phase 3.1 and 3.1.

Replace the frontend stored `gameState` by the latest obtained one.

---

## Acceptance Criteria

1. Given a `GameState` with two teams, and `activeTeam` is **X**, when `Unit` of **Team X** is clicked, the player should see `TurnCommandPanel` with available action according to that `Unit`.
2. Given a `GameState` with two teams, and `activeTeam` is **Y**, when `Unit` of **Team X** is clicked, the player should **NOT** see `TurnCommandPanel`. 
3. Given Player clicks `Move`, `grid` should render `allowedTiles` when there is any available.
4. Given Player clicks one of the `allowedTiles` and confirmed the action for `Move`, `Unit` should move to the target `coordinate`.
5. Given Player clicks `Bomb`, `grid` should render `allowedTiles` when there is any available.
6. Given Player clicks one of the `allowedTiles` and confirmed the action for `Bomb`, `Bomb` should be placed on the target `coordinate`.

## Log

Implementation issues found during the build (non spec gaps) are tracked in [match-p3-spec003-log.md](./match-p3-spec003-log.md).
