---
title: "Log: p3-spec008-match"
---

# Known Issues

Found while implementing the `Available Bombs` column of `MatchSummaryPanel` — not a gap in `p3-spec008-match.md` itself, but a backend bug the spec's formula depends on.

1. **`Unit.BombUsed` in `engine/match.go` was never maintained.** It was checked in `CommandPlaceBomb`'s placement guard (`match.go:127`) but never incremented on a successful placement, and never decremented when a bomb detonated. It stayed `0` for the whole match, so the frontend's `unit.maxBombCount - unit.bombUsed` formula always just evaluated to `maxBombCount`, regardless of what happened on the board.
   **Status: Solved.** `CommandPlaceBomb` now increments `unit.BombUsed` on a successful placement (`match.go:142`); `processChainDetonations` now decrements the owning unit's `BombUsed` exactly once per bomb at detonation (`match.go:329`), guarded with `max(owner.BombUsed-1, 0)` as a defensive floor.
