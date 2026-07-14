---
title: "Phase 3.6: Victory Cutscene and Rematch"
---

# Victory Cutscene and Rematch

## Context

Phase 3.5 implements the full game cycle of a match, but when the match concludes, the frontend didn't stop, resulting an error msg like `match already ended`. This spec adds a `VictoryCutscene` to conclude the match, and provide a way to restart the match.

## Goal

- Render `VictoryCutscene` when the match ends.
- Add `rematchButton` to restart a match.
- Initiate a plain `MatchSettingsScene`.
- Add `returnMatchSettingsButton` to return to `MatchSettingsScene`.

## Non-Goal

- Polished animations or tweens (easing curves, squash/stretch, particle effects, etc.).
- HUD / status panel.
- Detailed implementation of `MatchSettingsScene` - a rough page for scene entry is acceptable.

## Scene Entry

No change from spec001, except: `Rematch` (see [below](#rematch)) re-enters `MatchScene` via `scene.restart()` + `rematch()`, reusing the currently registered room, rather than a fresh scene launch from `DevBootScene`.

## Victory Cutscene

`VictoryCutscene` is used to:

- Present the victory result of this match.
- Provide options (buttons) for Players to redirect, e.g., rematch, back to match settings.
- Prevent Players from further interacting with the ended `MatchScene`.

### Additional Event in ResolveTurn

The match always concludes after resolving turn, i.e., `units` are killed when the `bomb` explodes. `resolveTurn()` includes `matchEndedEvent`, which tells the match concludes and who the winner is (or a draw).

`matchEndedEvent` should contain `winnerTeamId`. `winnerTeamId` means which team is the winner; `matchEndedEvent.winnerTeamId == -1` is for a draw game. A missing or out-of-range `winnerTeamId` is a client-side integration bug (not a normal user error): show it via the existing `ErrorPanel` and do not render `VictoryCutscene`.

Validate if:

- `winnerTeamId == -1 OR 1 <= winnerTeamId <= 2`, as currently the game only supports 2 players.

> Note:
>
> - `MatchEndedEvent` also contains `IsDraw`. Ignore this flag entirely — derive draw purely from `matchEndedEvent.winnerTeamId == -1`. (Removing `IsDraw` itself is a backend/engine change and out of scope for this frontend spec.)
> - Due to Go's JSON serialization, `winnerTeamId` could be omitted, but it shouldn't happen here because **0** means the match is still in progress. In theory `ResolveTurn()` should never return it.

`matchEndedEvent` must be handled **after** all `gameEvents` in `resolveTurn()` has been processed, i.e., all rendering, animation, tween ends. It renders `VictoryCutscene`. Details in next section.

### Visual Effect of Victory Cutscene

`VictoryCutscene` is a set of game objects added directly onto `MatchScene`'s display list, layered above the board via depth. `MatchScene` renders **2 major layers and 2 buttons**, sized to the full canvas (**100% width, 100% height**). Use the same fade-in effect as `TurnBanner`.

**All user interactions disabled except the buttons in `VictoryCutscene`.**

1. A dim background layer (semi-transparent scrim, consistent with `ConfirmDialog`'s dim background) covering **100% width, 100% height**. (Not a real blur filter — that would need the `filters-and-postfx` post-processing pipeline, a heavier lift than this spec's Non-Goal of polished effects calls for.)
2. A rectangle panel `VictoryBanner` which has a similar spec as `TurnBanner`: a **100% width, 144px height** rectangle banner. Font color is `0xffffff`.

- **For non-draw:**
  - Use `TEAM_COLORS` in `constants.ts`.
  - `Winner... Player {X}!` is rendered in the center of the banner in 2 lines. The font size of the 1st line is **36px** while the 2nd line is **48px**. A rough alignment is shown below.
  ```text
  +--------------------------+
  |     Winner...            | <- slightly left shift
  |         Player {X}!      | <- in the center
  +--------------------------+
  ```
- **For draw game:**
  - Fill with `0x4c4c4c`.
  - `Draw Game` is rendered in the center of the banner. Font size is **48px**.

3. A `rematchButton` button, same dimensions and coloring as `resolveButton` (`RESOLVE_BUTTON_WIDTH`/`RESOLVE_BUTTON_HEIGHT`, `PANEL_BUTTON_FILL_COLOR`, `PANEL_BUTTON_BORDER_COLOR` in `constants.ts`), with the text `Rematch`. This button should be placed underneath `VictoryBanner`. [Details for button handler](#rematch).
4. A `returnMatchSettingsButton` button, same dimensions and coloring as `rematchButton`, with the text `Return to Match Settings`. This button should be placed underneath `rematchButton`. [Details for button handler](#return-to-match-settings).

The buttons should be rendered **2s** after the rest of `VictoryCutscene` is rendered. No fade-in effect needed.

## Rematch

`MatchScene` fades out in **200ms**, then restarts itself (`scene.restart()`, no `LoadingScene` transition — consistent with this spec's Non-Goal excluding polished transitions) and fades back in over **200ms**, calling `rematch()`, which reuses the currently registered room.

> Note: `scene.restart()` can leave an in-flight fetch from the pre-restart scene instance (e.g. `create()`'s `getMatchState()` call, see `match-p3-spec002-log.md` issue #1) resolving after shutdown. Its callback must not mutate the new scene instance's display list — guard it (e.g. bail out if the scene has been shut down/restarted since the fetch started).

## Return to Match Settings

Create a `MatchSettingsScene` with a title `Match Settings`. This is just a stub scene. The concrete one will be implement in `stage-p3-spec001.md`.

`MatchScene` fades out in **200ms**, and transits to `MatchSettingsScene`.

The 200ms fade-out and `deleteMatch()` start at the same time. The transition to `MatchSettingsScene` waits for **both** to finish — the fade-out completing and the `deleteMatch()` request settling (success or failure) — so the fade is never cut short and `MatchSettingsScene` never appears before the delete attempt has run its course. A failing `deleteMatch()` delays the transition until it settles, but never blocks it permanently: log the error (and its reason) via `console.error` and proceed to `MatchSettingsScene` regardless of outcome.

---

## Acceptance Criteria

1. Given `resolveTurn()` contains `matchEndedEvent` with `winnerTeamId == 1`, `VictoryCutscene` should be render in blue and display the information that Player 1 wins the match.
2. Given `resolveTurn()` contains `matchEndedEvent` with `winnerTeamId == 2`, `VictoryCutscene` should be render in red and display the information that Player 2 wins the match.
3. Given `resolveTurn()` contains `matchEndedEvent` with `winnerTeamId == -1`, `VictoryCutscene` should be render in grey and display the information that the game is draw.
4. Given `VictoryCutscene` appears, Player cannot click anything except the buttons in `VictoryCutscene`.
5. When Player clicks `rematchButton`, the match should be restarted, with the same `gameCfg`.
6. When Player clicks `returnMatchSettingsButton`, the match should be deleted, with a smooth transition to `MatchSettingsScene`.
7. Given `resolveTurn()` contains `matchEndedEvent` with a missing or out-of-range `winnerTeamId` (i.e. not `-1` and not `1 <= winnerTeamId <= 2`), `ErrorPanel` should show the error and `VictoryCutscene` should not be rendered.
8. Given `deleteMatch()` rejects while transitioning to `MatchSettingsScene`, the error should be logged and `MatchSettingsScene` should still load.
