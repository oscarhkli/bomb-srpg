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
- A VS CPU option in Battle Mode

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
- `Back`: navigates up a layer, i.e., the sub-menu disappears and `Start Game` is restored.

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

- If Player clicks `Story Mode`, follow how `MatchSettingsScene` calls `startMatch()`:
  - Call `createMatchRoom()` to get `roomId`.
  - Use `roomId` to `initRoom()`.
  - Use `roomId` and the Prologue `GameCfg` above to call `createMatch()`.
  - `fadeTransition` to `MatchScene` with `roomId` and `playerTokens`.
- If `createMatchRoom()` or `createMatch()` fails, report the error and stay on `TitleScene` (no transition to `MatchScene`).
- Extract `startMatch()` so that it's reusable for both situations instead of blindly duplicating it. The extraction must preserve three properties the current implementation already guarantees: a re-entrancy guard so a second click can't start a second match, the fade-out and match-creation running concurrently with the transition waiting on both, and a failure path that fades back in and reports the error in place.

## Match Scene for Prologue Match

### Sprite

Boss is one of the Unit, but with a difference - since it's a NPC, Blue version is unavailable, but Red only (by design P2 is either Human or CPU). Team formation rules place a Boss on P2 only, so a Blue lookup can never occur. The other things follows the same as other units. See [p4-spec001-sprites](p4-spec001-sprites.md#sprites) for the ground rules.

| Entity         | Texture Key       | Path (relative to `sprites/`)    | Type             |
| -------------- | ----------------- | -------------------------------- | ---------------- |
| Prologue (Red) | unit_prologue_red | units/Prologue-Red.png (+ .json) | atlas (aseprite) |

These loads belong in `MatchScene.preload()`, added to the existing `SPRITE_MANIFEST`, and the archetype-to-texture lookup gains a `Prologue` entry for team 2.

### VS CPU

`gameCfg.vsCpu = true` marks a VS CPU match. The turn opens identically to VS Human — `startTurn()` in `beginTurn()`, then `SuddenDeathCutscene` (if needed) and `TurnBanner` — but when `gameState.activeTeam === 2` the CPU's whole turn arrives as data instead of as click events.

Interactions are locked from the CPU turn's `beginTurn()` and the Player regains control only after the next `TurnBanner` — their own turn's — has finished. `MatchScene` is hot-seat, so without this lock the Boss would be selectable while `activeTeam === 2`.

#### Polling for the CPU's turn

Once `startTurn()` resolves and `gameCfg.vsCpu && gameState.activeTeam === 2`, start calling `consumeCpuStatus()`. This runs **concurrently** with the `SuddenDeathCutscene → TurnBanner` rendering. The rendering owns the clock, the polling owns the data, and neither waits on the other.

`consumeCpuStatus()` typically does not return until the CPU has finished planning, so it behaves as a short long-poll and the first call usually answers `TurnPhaseReady`. `TurnPhasePlanning` is still possible and must be retried. Poll accordingly:

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

- **`turn` advanced and `activeTeam` is back to 1** — the CPU's turn committed on the server and only the animation was lost. Re-render the board from the fetched state and continue to the next turn. Mask the re-render with a fade-out → fade-in, as `fadeTransition` does for a Page swap. No error surfaced to the Player.
- **`turn` unchanged** — the server-side turn never completed. Surface an error and the match cannot proceed.

The recovery `getMatchState()` has no separate timeout — if the CPU turn is still running it resolves once that finishes, and reports the advanced turn.

Do not force-end the match on a poll timeout. The CPU's moves are committed to `TrueState` before `consumeCpuStatus()` can ever see them, so a lost batch is cosmetic, not a desync.

#### `TurnPhaseIdle` is an error, not a fallback

`startTurn()` sets the phase to `TurnPhasePlanning` under the room lock *before* it responds, so a poll issued after `startTurn()` resolves can only observe `TurnPhasePlanning` or `TurnPhaseReady`. Observing `TurnPhaseIdle` during a CPU turn means the mailbox was already drained by a second, competing consumer — the events are gone and will not reappear.

Treat it as a bug signal: stop polling, run the `getMatchState()` recovery above, and log it. Never render a `TurnPhaseIdle` response's events; they are always empty. The practical guard is to ensure only one poll loop is ever in flight per turn.

> Note: `/resolve` concatenates the two slices from `ServerStateManager.ResolveTurn`, so `/resolve` and the VS Human path are unchanged by this spec. `MatchScene`'s human branch still drops its own planning events through `resolveTurnPlayer`'s per-type filter, which is valid only while each `GameEvtType` belongs to exactly one phase. Splitting that path is tracked in [p4-spec009-match](p4-spec009-match.md) and is trigger-gated, not scheduled.

## Victory Cutscene for Prologue Match

When Prologue Match ends, it should return to `TitleScene` instead of `MatchSettingsScene`.

In a Prologue Match, `VictoryCutscene`'s lower button reads `Return to Title` and `fadeTransition`s to `TitleScene`; `Rematch` is unchanged and replays the Prologue in the same room. Styling, dimensions, and the `deleteMatch()`-and-fade handling follow [Return to Match Settings](p3-spec006-match.md#return-to-match-settings).

`gameCfg.vsCpu` is the discriminator — it is true exactly when the match was entered from Story Mode. Revisit if Battle Mode ever gains a CPU option.

---

## Acceptance Criteria

### Story Mode entry

1. Given `TitleScene`, when Player clicks `Start Game`, then `Start Game` disappears and `Story Mode` / `Battle Mode` / `Back` appear at the same position, sharing its font and hover Bomb icon.
2. Given the sub-menu is open, when Player clicks `Back`, then `Start Game` is restored.
3. Given the sub-menu is open, when Player clicks `Story Mode`, then a match is created with the Prologue `GameCfg` and `MatchScene` opens with the returned `roomId` and `playerTokens`.
4. Given Player clicks `Story Mode`, when `createMatchRoom()` or `createMatch()` fails, then the error is reported and the Player stays on `TitleScene`.
5. Given a Prologue Match has loaded, when the board renders, then the P2 Boss uses the `unit_prologue_red` texture.

### VS CPU turn

6. Given a Prologue Match and `activeTeam === 2`, when the turn opens, then `SuddenDeathCutscene` and `TurnBanner` play at their normal speed, the CPU's animation begins as soon as the banner finishes, and the Player cannot plan or select units until their own next `TurnBanner` has finished.
7. Given a `TurnPhaseReady` response and a finished `TurnBanner`, when the CPU turn renders, then `planGameEvents` animate first, then a 600ms hold, then `resolveTurnGameEvents`.
8. Given `planGameEvents` is empty, when the CPU turn renders, then the 600ms hold is skipped.
9. Given `resolveTurnGameEvents` ends with `matchEnded`, then `VictoryCutscene` opens; otherwise `beginTurn()` runs and the Player's turn opens.
10. Given polling exhausts its budget, when `getMatchState()` shows `turn` advanced and `activeTeam === 1`, then the board re-renders from that state and play continues with no error surfaced; when `turn` is unchanged, then an error is surfaced. The match is never force-ended.
11. Given a Prologue Match has ended, when `VictoryCutscene` shows, then the lower button reads `Return to Title` and `fadeTransition`s to `TitleScene`, and `Rematch` still restarts the Prologue in the same room.

### VS Human regression

12. Given a Battle Mode match (`gameCfg.vsCpu === false`), when a turn opens, then the Player can plan immediately after `TurnBanner` — no CPU polling delay, and no plan/resolve animation split.
13. Given a Battle Mode match, when `/resolve` returns, then its `gameEvents` render as one continuous sequence, unchanged from Phase 4.7.
14. Given a Battle Mode match has ended, when `VictoryCutscene` shows, then the lower button reads `Return to Match Settings` and returns to `MatchSettingsScene`.
