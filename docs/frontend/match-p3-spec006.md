---
title: "Phase 3.6: Victory Cutscene and Rematch"
---

# Victory Cutscene and Rematch

## Context

Phase 3.5 implements the full game cycle of a match, but when the match concludes, the frontend didn't stop, resulting an error msg like `match already ended`. This spec adds a `VictoryCutscene` to conclude the match, and provide a way to restart the match.

## Goal

- Render `VictoryCutscene` when the match ends.
- Add `rematchButton` to restart a match.

## Non-Goal

- Polished animations or tweens (easing curves, squash/stretch, particle effects, etc.).
- HUD / status panel.
- Detailed implementation of `MatchSetupScene` - a rough page for scene entry is acceptable. Detailed part will be initiated in `stage-p3-spec001.md`.

## Scene Entry

No change from spec001.

## Victory Cutscene

`VictoryCutscene` is used to:

- Present the victory result of this match.
- Provide options (buttons) for Players to redirect, e.g., rematch, back to match settings.
- Prevent Players from further interacting with the ended `MatchScene`.

### Additional Event in ResolveTurn

The match always concludes after resolving turn, i.e., `units` are killed when the `bomb` explodes. `resolveTurn()` includes `matchEndedEvent`, which tells the match concludes and who the winner is (or a draw).

`matchEndedEvent` should contain `winnerTeamId`. `winnerTeamId` means which team is the winner; `matchEndedEvent.winnerTeamId == -1` is for a draw game. A missing or out-of-range `winnerTeamId` is a client-side integration bug (not a normal user error): show it via the existing `ErrorPanel` and do not render `VictoryCutscene`.

Validate if:
- `-1 <= winnerTeamId <= 2`, as currently the game only supports 2 players.

> Note:
> - `MatchEndedEvent` also contains `IsDraw`. Ignore this flag entirely — derive draw purely from `matchEndedEvent.winnerTeamId == -1`. (Removing `IsDraw` itself is a backend/engine change and out of scope for this frontend spec.)
> - Due to Go's JSON serialization, `winnerTeamId` could be omitted, but it shouldn't happen here because **0** means the match is still in progress. In theory `ResolveTurn()` should never return it.

`matchEndedEvent` must be handled **after** all `gameEvents` in `resolveTurn()` has been processed, i.e., all rendering, animation, tween ends. It renders `VictoryCutscene`. Details in next section.

### Visual Effect of Victory Cutscene

`MatchScene` renders a **full screen** `VictoryCutscene`, in 2 major layers, and 2 buttons. Use the same fade in effect as `TurnBanner`.

**All user interactions disabled except the buttons in `VictoryCutscene`.**

1. A background layer that blurs the whole canvas: **100% width, 100% height**.
2. A rectangle panel `VictoryBanner` which has a similiar spec as `TurnBanner`: a **100% width, 144px height** rectangle banner. Font color is `0xffffff`.
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
    - Fill with `0xfcfcfc`.
    - `Draw Game` is rendered in the center of the banner. Font size is **48px**.
3. A `rematchButton` button, same dimensions and coloring as `resolveButton` as of today, with the text `Rematch`. This button should place underneath `VictoryBanner`. [Details for button handler](#rematch).
4. A `returnMatchSettingsButton` button, same dimensions and coloring as `rematchButton` as of today, with the text `Return to Match Settings`. This button should place underneath `rematchButton`. [Details for button handler](#return-to-match-settings).

The buttons should be rendered **2s** after the rest of `VictoryCutscene` is rendered. No fade-in effect needed.

> Note:
> - Agent should correct my wordings in this section with Phaser terms.
> - Is `VictoryCutscene` a screen instead? But now we render it on top of `MatchScene`. We can change this representation though.

## Rematch

The whole `MatchScene` should fade out in **200ms**. A new `MatchScene` should fade in in **200ms**. The new `MatchScene` should use the **same** set of `gameCfg` and `roomId` to create match.

> Note: Agent should tell if this is a legit way to work on mismatch as this is the 1st time writing spec about scene entry. Should we instead use a `LoadingScene` as a transition?

## Return to Match Settings

`MatchSettingsScene` has not started yet, so the moment we can't transit to the scene. A `console.log` can be substituted as a placeholder, or create a stub `MatchSettingsScene` with just a title `Match Settings`. 

> Note: Agent should decide which way is better.

## Pending Backend Gaps

Currently there are only 2 ways to remove a match:
1. Surrender - it removes the `MatchRoom`.
2. Match is idled for a long time and cleaned up by `StartCleanupLoop()`.

It's impossible to implement rematch as the backend blocks the match creation in a MatchRoom that contains a match, unless either:
1. Unless there's an API in backend to remove a match.
2. Enhance `CreatMatch()` to remove the existing match inside a the `MatchRoom`.

> Note: Agent should suggest whether the spec should be split into 1) Rendering and 2) Button handler for rematch.

---

## Acceptance Criteria

1. Given `resolveTurn()` contains `matchEndedEvent` with `winnerTeamId == 1`, `VictoryCutscene` should be render in blue and display the information that Player 1 wins the match.
2. Given `resolveTurn()` contains `matchEndedEvent` with `winnerTeamId == 2`, `VictoryCutscene` should be render in red and display the information that Player 2 wins the match.
3. Given `resolveTurn()` contains `matchEndedEvent` with `winnerTeamId == -1`, `VictoryCutscene` should be render in grey and display the information that the game is draw.
4. Given `VictoryCutscene` appears, Player cannot click anything except the buttons in `VictoryCutscene`.
5. When Player clicks `rematchButton`, the match should be restarted, with the same `gameCfg`.
