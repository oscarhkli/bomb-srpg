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

- Polished animations or tweens.
- HUD / status panel.
- Player can click and view the information of `unit` and `bomb`.
- Warning message about data out-of-sync with backend and frontend.

## Scene Entry

No change from spec001.

## Notes on Coloring

All the color HEX below will be adjusted in future to align with pixel style palettes. Always define in `constants.ts` for better maintenance.

## User Interaction and Data Fetching

This spec consist of various user interactions. Some of them requires data fetching. Some of them submit action to server and react based on the resopnse.

### TurnCommand Panel

`TurnCommandPanel` is rendered as **192Wx144Hpx** rectangle when needed (to be explained later). It's transparent, with no border. This contains the actions available for Player so choose. To simplify in the early stage. There are 3 buttons in the panel, `moveButton`, `placeBombButton`, `backButton`, distributed in 2 rows as illustrated below:

```text
+-------------+
| Move   Bomb |
|        Back |
+-------------+
```

All 3 buttons are rendered as **92Wx64Hpx** pill-shape, filled with `0x583f0e` and ocpacity **20%**, with **8px** `0xdc9e23` border color ocpacity **100%**. Text color is also the same color and opacity as border. Font color is `Roboto`.

> Note: The dimensions of buttons are rough numbers. During the implementation, iterations might needed so that the numbers should be adjust to fit with the real scenario, update the specs and remove this line.

### Store the Latest `gameState`

In the upcoming sections, the user interaction depends on the `gameState` obtained in `getMatchState()`. There should be a way to store and refresh it when needed. See the next sections.

> Note: Agent should determine how `gameState` should be kept in the frontend.

### User Interaction

`TurnCommandPanel` can only be shown and enabled when all the below situations are satified:

1. A `unit` is clicked.
2. `unit.team` equals to `state.activeTeam`.

Violating either means the unit is read-only. It prints a `console.log()` just as the current phase.

Additionally:

- If `unit.hasMoved` is **true**, `moveButton` should be disabled and rendered with HEX `0x999999`.
- If `unit.hasUsedSkill` is **true**, `placeBombButton` should be disabled and rendered with HEX `0x999999`. 

> Non-goal note:  
> Since currently we're working on Phase 3, where the game is pass-and-play. Both players are in the same client browser tab. For multiplayer online more, additional rules should be add for displaying and enabling `TurnCommandPanel`: The current client's team equals to `state.activeTeam`. 

### Move

If Player clicks `moveButton`, the frontend should call backend via `getAllowedTiles()`, using the payload:

```json 
{
  unitId: ${unit.id},
  turnCmdType: "move"
}
```

If non-200 HTTP result returns, display the errorMessage like how we handle Network error.

If 200 HTTP result returns with `AllowedTilesResponse`, these are the coordinates of the `tiles` that `unit` can move. For all matched `tiles`:

- Add a layer of HEX `0x86c64f with opacity **30%** layer on top of it. This adds a "glowing effect" to those `tiles`.
> Note: To be reviewed with Agent - there could be a better terminology in phaser to describe this visual effect.
- Add a click handler: When clicked,
  - The border of the glowing layer should change to `0xdaedca` with opacity **100%**.
  - Confirm if Player wants to execute move (Details in next section).
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

### Place Bomb

The mechanism `placeBomb` is similar to `move`. To avoid repeating (unless Agent thinks we should explicitly repeat the section):

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

`actionStack` should be cleared and `TurnCommandPanel` should be hidden when Player does not interact with the same unit, e.g., clicking other units, resetting turn (in future phase), etc..

### Performance consideration

There might be perfomance issue regarding `getAllowedTiles()` since each click calls API once, but the result mostly doesn't change. We might have to consider caching mechanism, or use an alternative data fetching, e.g., call `getAllowedTiles()` for both `move` and `placeBomb` when a unit is first clicked after each `TurnCommand` is executed.

## Follow-up Actions after TurnCommand Submission

`TurnCommandPanel` should be closed no matter if `submitTurnCommand()` is successful or not.

If non-200 HTTP result returns, display the errorMessage like how we handle Network error.

The below states the follow-up action when 200 HTTP result returns

`submitTurnCommand()` **will** receive `gameEvents` instead of `gameState` before the implementation of this spec.

Currently we focus on `move` and `placeBomb` action. If there is any `gameEvent` with `type` other than `unitMoved` or `bombPlaced`, log it with `console.log()`.

### Visual Effect for `unitMovedEvent`

`unitMovedEvent` should contain at least `unitId`, `from` and `to`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `unitMovedEvent.unitId` equals to `unitId` of current actor.
- `grid[unitMovedEvent.from.y][unitMovedEvent.from.x].occupantUnit` and `occupantId` matches `unitMovedEvent.unitId` , i.e., the chosen `unit` is still in the `from` position.
- `grid[unitMovedEvent.to.y][unitMovedEvent.to.x].OccupantNone`, i.e., can be occupied by the chosen `unit`.
- `unitId` exists in `gameState.units` and `position` matches with `unitMovedEvent.from`.

> To be discussed:
> 1. Are some of the validations redundant? 
> 2. Should we smoothen the move animation here using the **Manhattan Movement** path, or just to move in a straight path and polish in future?

### Visual Effect for `bombPlacedEvent`

`bombPlacedEvent` should at least contain `bombId`, `unitId`, `position` and `countdown`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `bombPlacedEvent.unitId` equals to `unitId` of the chosen `unit`.
- `grid[bombPlacedEvent.position.y][bombPlacedEvent.position.x].OccupantNone`, i.e., can be occupied by the new `bomb`.

> To be discussed:
> 1. Are some of the validations redundant? 

## Refresh Final Sanity Check

After all `gameEvents` are handled. call `getMatchState()` once for sanity check. If `grid` looks different with `gameState` expected, flag it in errorMessage, then re-render the `tiles`, `units` and `bombs` using new `gameState` as the same way as Phase 3.1 and 3.1.

Replace the frontend stored `gameState` by the latest obtained one.

---

## Acceptance Criteria

1. Given a `GameState` with two teams, and `activeTeam` is **X**, when `Unit` of **Team X** is clicked, the player should see `TurnCommandPanel` with available action acccording to that `Unit`.
2. Given a `GameState` with two teams, and `activeTeam` is **Y**, when `Unit` of **Team X** is clicked, the player should **NOT** see `TurnCommandPanel`. 
3. Given Player clicks `Move`, `grid` should render `allowedTiles` when there is any available.
4. Given Player clicks one of the `allowedTiles` and confirmed the action for `Move`, `Unit` should move to the target `coordinate`.
5. Given Player clicks `Bomb`, `grid` should render `allowedTiles` when there is any available.
6. Given Player clicks one of the `allowedTiles` and confirmed the action for `Bomb`, `Bomb` should be placed on the target `coordinate`.
