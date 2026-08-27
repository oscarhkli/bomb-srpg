---
title: "Phase 4.8: Add Story Mode to TitleScene and Initiate Prologue Match"
---

# Phase 4.8: Add Story Mode to TitleScene and Initiate Prologue Match

## Context

This spec introduces Prologue of Story Mode in order to enable the first VS CPU Match in the game.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

> Note: Unless specified, all vector graphic components are temp representation and will eventually be replaced by pixel art sprites. The dimensions and alignments are just rough estimations - accept they aren't perfect, or even slightly misplaced and out of range.

## Goal

- `TitleScene` allows Players to choose between Story Mode and Battle Mode (the existing Local Match).
- Human player interacts with CPU player in `MatchScene` through Story Mode.

## Non-Goal

- Complete Story Mode

## Scene Entry

- `MatchScene` for Story Mode is launched by `TitleScene` after a successful `createMatch()` with a specific parameter.
- `MatchScene` for Story Mode returns to `TitleScene` after Player clicks the button in `VictoryCutscene`.

---

## Title Scene

New option is added to `TitleScene`.

In `GameModeSelectionPanel`, `Start Game` no longer enters `MatchSettingsScene`. Instead, a nested sub menu should be presented like traditional arcade game.

```text
Start Game
-> Story Mode
-> Battle Mode
-> Back
```

- When Player clicks `Start Game`, `Start Game` should disappear. The sub-menu, "Story Mode, Battle Mode, etc." should appear next at the same position.
- For simplicity, all 3 options should share the same property as `Start Game`, i.e., font size, font family, hover with a Bomb icon.
- `Story Mode`: `fadeTransition` `MatchScene` for Prologue Match (details in next section).
- `Battle Mode`: `fadeTransition` `MatchSettingsScene` as of `Start Game` prior Phase 4.7.
- `Back`: navigates up a layer, i.e., displaying `Story Mode` only.

### Creating Prologue Match

Prologue Match has a predefined `GameCfg`. There isn't any option for Player to set. Once Player clicks `Story Mode`, `TitleScene` should do all the settings and enter `MatchScene`.

```json
{
  "vsCpu": true,
  "stagePreset": "Plain",
  "p1Slots": [
    {
      "archetype": "King",
      "role": "King"
    },
    {
      "archetype": "Fighter",
      "role": "Normal"
    },
    {
      "archetype": "Witch",
      "role": "Normal"
    },
    {
      "archetype": "Fighter",
      "role": "Normal"
    },
    {
      "archetype": "Bandit",
      "role": "Normal"
    }
  ],
  "p2Slots": [
    {
      "archetype": "Prologue",
      "role": "Boss"
    }
  ],
  "maxTurns": 30,
  "allowResetTurn": true
}
```

- If Player clicks `Story Mode`, follow how `MatchsScene` calls `startMatch()`:
  - Call `createMatchRoom()` to get `roomId`.
  - Use `roomId` to `initRoom()`.
  - Use `roomId` and `MatchSettingsScene.gameCfg` to call `createMatch()`.
  - `fadeTransition` to `MatchScene` with `roomId` and `playerTokens`.
- If `createMatchRoom()` or `createMatch()` fails, report the error and stay on `TitleScene` (no transition to `MatchScene`).
- Extract `startMatch()` so that it's reusable for both situations instead of blindly duplicating it.

## Match Scene for Prologue Match

### Sprite

Boss is one of the Unit, but with a difference - since it's a NPC, Blue version is unavaible, but Red only (by design P2 is either Human or CPU). The other things follows the same as other units. See [p4-spec001-sprites](p4-spec001-sprites.md#sprites) for the ground rules.

| Entity         | Texture Key       | Path (relative to `sprites/`)    | Type             |
| -------------- | ----------------- | -------------------------------- | ---------------- |
| Prologue (Red) | unit_prologue_red | units/Prologue-Red.png (+ .json) | atlas (aseprite) |

### VS CPU

`gameCfg.vsCpu = true` marks a VS CPU match. The turn opens identically to VS Human — `startTurn()` in `beginTurn()`, then `SuddenDeathCutscene` (if needed) and `TurnBanner` — but when `gameState.activeTeam === 2` the CPU's whole turn arrives as data instead of as click events.

#### Polling for the CPU's turn

Once `startTurn()` resolves and `gameCfg.vsCpu && gameState.activeTeam === 2`, start calling `consumeCpuStatus()`. This runs **concurrently** with the `SuddenDeathCutscene → TurnBanner` rendering. The rendering owns the clock, the polling owns the data, and neither waits on the other.

`consumeCpuStatus()` blocks server-side on the room lock while the CPU is planning, so it behaves as a short long-poll — the first call usually returns `TurnPhaseReady` directly. Poll accordingly:

- Fire the first call immediately, not after a delay.
- On `TurnPhasePlanning`, retry with backoff: `250ms → 500ms → 1s → 2s`, capped at 2s.
- Give up after **30s** total.

`TurnPhaseReady` is the only success. The response carries the CPU's turn as two arrays:

| Field                   | Contents                                                                          |
| ----------------------- | --------------------------------------------------------------------------------- |
| `planGameEvents`        | what the CPU did — `unitMoved`, `bombPlaced`                                       |
| `resolveTurnGameEvents` | what the board did in response — countdowns, explosions, damage, deaths, `matchEnded` |

#### Rendering the CPU turn

After the `TurnBanner` has finished **and** a `TurnPhaseReady` response is in hand:

1. Animate `planGameEvents` — the CPU's units move and place bombs, using the same animations a human's commands produce.
2. Hold for **600ms**. This is the beat that stands in for the human pressing Resolve; without it the CPU's move and the explosion it triggers read as one indivisible event.
3. Animate `resolveTurnGameEvents`.
4. If `resolveTurnGameEvents` ends with `matchEnded`, transition to `VictoryCutscene` instead of `beginTurn()`.
5. Otherwise the CPU's turn is over and `activeTeam` is back to 1 — call `beginTurn()`, exactly as the VS Human path does after `/resolve`. The next `startTurn()` opens the Player's turn.

If `planGameEvents` is empty (the CPU passed), skip step 2 — a pause with nothing before it reads as a stall.

#### When polling fails

Two distinct failures, distinguished by one `getMatchState()` call after the 30s budget expires:

- **`turn` advanced and `activeTeam` is back to 1** — the CPU's turn committed on the server and only the animation was lost. Re-render the board from the fetched state and continue to the next turn. Use `fadeTransition` to hide the re-render procees from the Player. No error surfaced to the Player.
- **`turn` unchanged** — the server-side turn never completed. Surface an error and the match cannot proceed.

Do not force-end the match on a poll timeout. The CPU's moves are committed to `TrueState` before `consumeCpuStatus()` can ever see them, so a lost batch is cosmetic, not a desync.

#### `TurnPhaseIdle` is an error, not a fallback

`startTurn()` sets the phase to `TurnPhasePlanning` under the room lock *before* it responds, so a poll issued after `startTurn()` resolves can only observe `TurnPhasePlanning` or `TurnPhaseReady`. Observing `TurnPhaseIdle` during a CPU turn means the mailbox was already drained by a second, competing consumer — the events are gone and will not reappear.

Treat it as a bug signal: stop polling, run the `getMatchState()` recovery above, and log it. Never render an `TurnPhaseIdle` response's events; they are always empty. The practical guard is to ensure only one poll loop is ever in flight per turn.

> Note: `ServerStateManager.ResolveTurn` concatenates the two slices, so `/resolve` and the VS Human path are unchanged by this spec. `MatchScene`'s human branch still drops its own planning events through `resolveTurnPlayer`'s per-type filter, which is valid only while each `GameEvtType` belongs to exactly one phase. Splitting that path is tracked in [p4-spec009-match](p4-spec009-match.md) and is trigger-gated, not scheduled.

## Visual Spec for Victory Cutscene for Prologue Match

When Prologue Match ends, it should return to `TitleScene` instead of `MatchSettingsScene`.

1. Create a new `returnTitleButton` button, same dimensions and coloring as `returnMatchSettingsButton` with the text `Return to Title`.
2. Swap `returnMatchSettingsButton` for `returnTitleButton`.
3. Refer to [Details for button handler](p3-spec006-match.md#return-to-match-settings) to add the click handler for `returnTitleButton` with one exception: `returnTitleButton` `fadeTransition` to `TitleScene`.

---

## Acceptance Criteria

1. Given `MatchScene` has loaded a match, when the unit is clicked, then all buttons in `TurnCommandPanel` display their pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
2. Given a button in `TurnCommandPanel`, when the button is disabled due to some reasons, then the button with the label should turn in its greyscale color.
3. Given a button in `TurnCommandPanel`, when the button is not disabled and selected, then the button with the label should change the sprite to mimic glow effect.
4. Given a button in `TurnCommandPanel`, when the button is not disabled and clicked, then the button with the label should change the sprite to mimic click effect, and label should shift **2px** downwards.
5. Given `MatchScene` has loaded a match, when `ConfirmDialog` pops up, then all buttons in `ConfirmDialog` display their pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
6. Given a button in `ConfirmDialog`, when the button is not disabled and selected, then the button with the label should change the sprite to mimic glow effect.
7. Given a button in `ConfirmDialog`, when the button is not disabled and clicked, then the button with the label should change the sprite to mimic click effect, and label should shift **2px** downwards.
