---
title: "Phase 3.11: MatchScene Render-Path Cleanup"
---

# MatchScene Render-Path Cleanup

## Context

Deferred out of `match-p3-spec005.md` so the full game cycle lands first, regardless of
performance. Open questions carried over from that spec's Game Loop section:

1. Since we have no plan to change tileType in the mid-game, should we render the grid in the beginning instead re-rendering everytime in `renderBoard()`? The current flow does redundant rendering.
2. Since we trust what the backend provides, the frontend mostly only do the rendering and player's interaction, is current sanity check really necessary, or YAGNI?
3. If sanity check isn't necessary, do we really need to re-render every time when we refresh `gameState`?

### Why the Game Loop calls `getMatchState()` twice

The two calls have different jobs, which is the concrete ground for Q1/Q3:

- The **initial** call (before the loop) is the *only* full-board paint — it renders `grid` + `occupants` so there is something on screen before the loop begins. Without it, nothing is drawn.
- Every **per-turn** call inside the loop is a *refresh* — it repopulates the interaction maps for the player's turn and hands a fresh `gameState` snapshot to the subsequent `resolveTurn` events.

So if per-turn re-render is dropped, only the initial call paints the board wholesale; the per-turn calls just refresh the maps and feed the event handlers. That is the tradeoff Q1/Q3 are weighing.

## Non-Goal

- Any gameplay or turn-lifecycle behavior change.
