# p4-spec003-sprites — Known Issues

## refreshTurnPanelIfReady() shared tryRenderStageBackground()'s stale-gameCfg race

**Status:** Solved

`MatchScene.tryRenderStageBackground()` needed a `gameCfgLoaded`/`gameStateLoaded` flag pair
because `this.gameCfg`/`this.gameState` are never cleared on scene re-entry — a truthy check
alone can pass using the previous match's stale value while the fresh fetch for the new entry is
still in flight (found and fixed during this spec's code review; see git history for
`MatchScene.ts`).

`refreshTurnPanelIfReady()` (pre-existing, untouched by this spec) had the identical shape,
gating on `this.gameState && this.gameCfg`'s truthiness rather than a per-entry loaded flag. On
a rematch where `getMatchState()` resolves before the fresh `getMatchConfig()`, this could
briefly call `turnPanel.update()` with the *previous* match's stale `maxTurns`, corrected once
the real `getMatchConfig()` resolved.

**Remark:** Not introduced by this spec — TurnPanel display, unrelated to Stage backgrounds —
but cheap to fix once `gameStateLoaded`/`gameCfgLoaded` existed: switched the same guard to
those flags instead of raw truthiness. No spec content added — this is a TurnPanel/`maxTurns`
correctness fix, not a Stage-background behavior, so it stays out of `p4-spec003-sprites.md`'s
Goal/Acceptance Criteria.

## No terrain/occupant depth interleaving — Stage background is one flat, undivided sprite

**Status:** Deferred

`p4-spec001-sprites.md`'s Occupant Depth rule sorts Units/SoftBlocks/Bombs by row so a lower-row
occupant renders in front of a higher-row one. The Stage background can't participate in that
scheme at all: Phaser's `Depth` component is a single scalar per GameObject, and the background
is one `Image` covering the whole grid — there's no way to vary depth within a single sprite by
Y position.

This doesn't break anything `p4-spec003-sprites.md` promises: AC3 only requires the background
to render behind *all* occupants unconditionally, which `DEPTH_STAGE_BACKGROUND (0) <
DEPTH_OCCUPANT (10+)` already guarantees regardless of row. The `Standard` stage's `TerrainBlock`
art stays within its own tile's bounds today, so nothing visually breaks yet.

**Remark:** This becomes a real gap the moment terrain art gets a vertically-tall feature —
`TerrainTower` already exists as an engine terrain type, and a tower should occlude a unit
standing behind it the way an occupant does. Neither p4-spec001 nor p4-spec003 ever promised
terrain-vs-occupant depth interleaving, so this isn't a defect in either — it's new capability
that needs its own spec once tall terrain art exists to design against.

Two viable directions for that future spec, both keeping depth per-row (not per-tile, which
would fracture any element straddling a tile's diagonal corner into four pieces):
- Runtime crop: `Image.setCrop(x, y, width, height)` sliced into one full-width, `TILE_SIZE`-tall
  horizontal strip per grid row (edge strips absorbing the decorative bleed above row 0 / below
  the last row), each strip depth-tagged via the same `occupantDepth()` formula.
- Asset-level slicing: Aseprite's Slice tool exporting named sub-regions as separate atlas
  frames, each loaded as an independent row-`Image`.

p4-spec001's own Non-Goal list deferred Tilemap adoption to "next phase" — worth noting a
`TilemapLayer` is also one GameObject with one depth, so it doesn't solve this for free either;
it would need the same per-row-layer split as the options above.
