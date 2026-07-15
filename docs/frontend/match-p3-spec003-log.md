---
title: "Log: match-p3-spec003"
---

# Known Issues

Found via a `frontend-code-review` pass after implementation, not by unit tests. These are bugs in the client-layer implementation, not gaps in `match-p3-spec003.md` — logged here for traceability since that spec is what surfaced them. All three were fixed in the same pass, test-first via `/tdd`.

1. **`MatchScene.onUnitClicked` had no `confirmDialog.isOpen` guard**, unlike every other entry point that can disturb panel/dialog state (`onBackButtonClick`, `onActionButtonClick`, the allowed-tile pointerdown handler). Since `ConfirmDialog`'s dim background isn't interactive, any active-team unit stayed clickable while a confirm was pending — clicking one silently discarded the pending Yes/No via `openFor` → `closeImmediately` → `hideConfirm`.
   **Status: Solved.** Added the same `isOpen` guard at the top of `onUnitClicked`.
2. **No re-entrancy guard around `handleTurnCommand`'s network round-trip.** `ConfirmDialog.hide()` fires synchronously on "Yes," so `isConfirmOpen()` goes `false` immediately, but `TurnCommandPanel.closeImmediately()` isn't called until `submitTurnCommand()` resolves — leaving the panel/overlay clickable and able to trigger a second concurrent submission during that window.
   **Status: Solved.** Added a private `isSubmitting` flag, set before `submitTurnCommand` and cleared in a `finally` block.
3. **`applyUnitMoved`'s tween set an absolute per-move delta (`toCenter - fromCenter`) instead of a cumulative offset (`g.x + delta`).** Since `renderBoard()` is now only called on grid mismatch (not on every successful turn), the same `Graphics` object persists across moves — a unit moving twice without a mismatch would tween to the same offset both times, visually undoing the first move. Unreachable today (no commit/turn-advance flow exists yet in this spec — a unit's `hasMoved` blocks a second move), but latent for whenever turn-advance wiring lands.
   **Status: Solved.** Changed to `g.x + (toCenter.cx - fromCenter.cx)` / `g.y + (toCenter.cy - fromCenter.cy)`.
4. **`isSubmitting` (added to fix issue #2 above) only guards against a second `submitTurnCommand()`/`resolveTurn()` request actually firing — it does not disable `TurnCommandPanel` or unit-click interaction while a request is in flight.** Those remain clickable during the round-trip; a duplicate command is built/confirmed and then silently no-op'd rather than visibly blocked. `match-p3-spec007.md`'s interaction lock contract requires disabling *all* user interactions on click, not just no-op'ing a re-submission.
   **Status: To be resolved by match-p3-spec007.** Confirm during `/frontend-sdd` implementation of spec007 whether its interaction-lock contract gets applied back to `TurnCommandPanel`/unit-click handling here, or deferred further.

Two mechanical cleanups were made in the same pass, not logged as issues since they're refactors, not bugs:
- `handleTurnCommand`'s inline anonymous param type replaced with the existing `TurnCommand` interface (`types/api.ts`).
- `TurnCommandPanel`'s `overlayTiles` changed from `Map<string, Graphics>` (key never read) to `Graphics[]`.
