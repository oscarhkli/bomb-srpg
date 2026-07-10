---
title: "Phase 3.4: Render Resolve Turn"
---

# Render Resolve Turn

## Context

This spec adds turn system command `ResolveTurn` control that trigger the server calculation and render the outcomes based to the `gameEvents` received.

## Goal

- `MatchScene` renders `TurnPanel` showing the current turn.
- `MatchScene` renders `ResolveTurnButton`, allowing Player to end the turn.
- Player can end the turn and see the effects after the backend calculation, e.g., `unit` died, `bomb` detonated, etc..
- Determine the rendering sequence of `gameEvent` due to chain-reaction.

## Non-Goal

- `OptionPanel` Showing the summary and the system command buttons - Phase 3.5+.
- `ResetTurnButton` - Phase 3.5+.
- Polished animations (mild effect is fine).
- HUD / status panel.

## Layout

### Store the Latest `gameCfg`

`MatchScene` keeps the latest `gameCfg` as a private scene-instance field (same pattern already used for `roomId`/`playerTokens`, e.g. `private gameCfg!: GameCfg;`)

### Bomb / SoftBlock Graphics Lookup

`MatchScene` adds `bombGraphicsById: Map<number, Phaser.GameObjects.Graphics>` and `softBlockGraphicsById: Map<number, Phaser.GameObjects.Graphics>` private fields, mirroring the existing `unitGraphicsById` pattern. `renderBombs`/`renderSoftBlocks` populate these maps by `bomb.id`/`softBlock.id` as they create each `Graphics` object.

This is required so `gameEvent` handlers below (`bombCountdownUpdated`, `bombExploded`, `softBlockDestroyed`) can look up and mutate/destroy the specific rendered object for a given ID, instead of scanning `boardObjects`.

**Implementation note:** The `bombGraphicsById`/`softBlockGraphicsById`/`unitGraphicsById` maps are owned and mutated by `MatchScene`, but the event-sequencing/animation logic described in "ResolveTurn and the Subsequent Visual Effects" and "Rendering Sequence" below is implemented in a dedicated `web/src/rendering/` module (`resolveTurnPlayer.ts`, `blastEffects.ts`, `reachTime.ts`), not inline in `MatchScene`. `MatchScene` hands the resolved `gameEvents` batch and its graphics maps to `playResolveTurnEvents()`, which reads/mutates those maps and schedules all timing via `scene.time.delayedCall`. This keeps `MatchScene` focused on state/graphics ownership and UI wiring. Constants exclusive to this rendering module (blast/fire timing, colors, sizes, and their depth bands) live in a local `web/src/rendering/constants.ts` rather than the shared `web/src/constants.ts`, since they have no consumers outside that module.

### Turn Panel

`TurnPanel` is rendered as **96Wx48Hpx** rounded square panel at the top left hand corner of the `MatchScene`, leaving 48px space from the top and left edges. Its depth is same as `TurnCommandPanel`.

`TurnPanel` display 4 elements: A text `Turn`, `gameState.turn`, followed by a text `/`, then `gameCfg.maxTurns`. The rough sample display is shown below:

```text
+-----------+
| Turn      |
|    2 / 30 |
+-----------+
```

- `TurnPanel` should leave 8px padding on each side.
- The top `Header` section should fill depends on the `gameState.activeTeam`. Use `TEAM_COLORS` in `constants.ts`.
- The bottom `Value` section should be transparent. The whole text is right aligned, but to avoid the text shifting due to the number, 3 text should be split into independent elements and on the exact location. The maximum value of both numeric values is **99**.
- Font family is `GAME_FONT_FAMILY`, `0xeeeeee`. If `gameState.turn` > `gameCfg.maxTurns`, render `gameState.turn` in HEX `0xff0000` indicating the match is now in sudden death.

**Remarks:** The dimensions and positions are temporary values and will be adjust during the implemenatation.

### ResolveTurn Button

`MatchScene` should render `ResolveButton` **48px** below its top edge. This is a **320Wx72** center-aligned text `End this turn`. Font family is `GAME_FONT_FAMILY`, font color is `PANEL_BUTTON_BORDER_COLOR`.

A click handler should be added to `ResolveButton`, opening `ConfirmDialog` with text `Confirm to end this turn?`. If Player clicks `No`, `ConfirmDialog` will be closed. If Player clicks `Yes`, it will trigger `ResolveTurn`. If `TurnCommandPanel` currently has an open action (e.g. a unit's Move/Bomb panel with allowed tiles shown), clicking `ResolveButton` first resets it to closed/no-selection — `ConfirmDialog` for `ResolveTurn` should never appear on top of a stale, still-interactive `TurnCommandPanel` action stack.

**`ConfirmDialog` size update:** `ConfirmDialog` (shared component, originally spec'd in spec003 at 160Wx100Hpx) is enlarged to **240Wx144Hpx** — the longer resolve-turn confirmation text no longer fits within the original size. This affects `ConfirmDialog` globally, not just its use here.

### Error Panel

All `errorMessage`s referenced throughout this spec (validation failures, network errors, final sanity-check mismatches, etc.) render in a fixed **Error Panel**, not as a one-off centered text — a single overlapping error was unreadable, and multiple errors within one action (e.g. a network failure followed by a refresh failure) must each stay legible.

- Fixed position: same left margin as `TurnPanel` (`TURN_PANEL_MARGIN`), **16px** below `TurnPanel`'s bottom edge.
- Size: **240Wx400Hpx**, dark semi-transparent background (`0x1a1a1a` at 75% opacity), depth above every other UI layer.
- Pinned to the camera viewport (`setScrollFactor(0)`), like `TurnPanel`/`ResolveButton`/`ConfirmDialog`.
- Each new error message is appended as its own word-wrapped line below the previous ones (not overlapping), padded **8px** from the panel edges.
- The panel and its accumulated messages are cleared at the start of each new user-initiated action (a turn-command submission or a `ResolveTurn`) — not inside the board-refresh path itself, since some flows (e.g. the final sanity-check mismatch) call `showError` immediately before a synchronous board re-render, and clearing there would destroy the message before it's ever seen.

## ResolveTurn and the Subsequent Visual Effects

After the Player confirms to end the turn, frontend should call the backend via `resolveTurn()`. Just as the other APIs, it request `roomId` and correct `playerToken` to operate. Also, print the errorMessage if there is.

If the API is correctly executed, the backend will do to all the heavy lift, and return a series of `gameEvents`. 

### BombCountdownUpdatedEvent

`bombCountdownUpdatedEvent` should at least contain `bombId` and `countdown`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `bomb` with `bombCountdownUpdatedEvent.bombId` exists in `gameState.bombs`.

For each `bombCountdownUpdatedEvent`: 

1. Re-render the number shown in the `bomb` using `bombCountdownUpdatedEvent.countdown`. If countdown is **0**, render a `!` with HEX `0xff0000` instead.

### BombExplodedEvent

`bombExplodedEvent` should at least contain `bombId` and `affectedPositions`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `bomb` with `bombExplodedEvent.bombId` exists in `gameState.bombs`.
- `grid` contains all the coordinates in `affectedPositions`, i.e., no out-of-bound problem.

This event renders a cardinal-ray blasting effect of a `bomb`. For each `bombExplodedEvents`:

1. Look up the `bomb's` position.
2. Remove the `bomb` image from the `grid`.
3. The animation should start from `bomb's` position, extending its blast outward in 4 directions simultaneously. Tile `T` is reached at `reachTime(bombPosition, T) = distance(bombPosition, T) × BLAST_SPEED_MS_PER_TILE`, a new constant in `web/src/rendering/constants.ts` (placeholder value — tunable later without touching the sequencing rules below).
    - Each of the 4 cardinal rays renders as a single beam that elongates outward over time (not per-tile flashes), reaching each tile at that tile's `reachTime`. The beam's fixed perpendicular width (the non-elongated dimension) is **32px** — narrower than the 48px tile — centered on the bomb's row/column. The beam (and its growing head/tip) is **pill-shaped** (fully-rounded rect), not a hard-edged rectangle.
    - The blast should be rendered in 3-layer gradient color. The outermost starts with HEX `0xf58e27`, then `0xf5ee27`, to the innermost `0xfcfabb`. Opacity is **60%**. The gradient bands split the beam into thirds of *that direction's own* max blast length — a short-range ray is fully outer→inner across its short length, not truncated by an absolute-distance mapping shared across all rays.

Once the blast has finished growing, it lingers (burning) for `BLAST_DURATION_MS` (~3s, placeholder value) before fading out. This lingering tail does not block other unrelated `gameEvents` — independent bombs/occupant effects render concurrently during it.

### UnitDamagedEvent

`unitDamagedEvent` should at least contain `unitId` and `newHp`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `unit` with `unitDamagedEvent.unitId` exists in `gameState.units`.

For each `unitDamagedEvent`:

1. Look up the `unit's` position.
2. Render a fire shape on top of the blast and `unit`, representing the `unit` is burning. Size is **42px** (larger than the 32px blast beam, so it visibly overlaps). Opacity is **70%**.
3. If `unitDamageEvent.newHp > 0`, remove the fire after **5s**. Otherwise, let it burn until its `unitDiedEvent` being processed.

### UnitDiedEvent

`unitDiedEvent` should at least contain `unitId`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `unit` with `unitDiedEvent.unitId` exists in `gameState.units`.

For each `unitDiedEvent`:

1. Look up the `unit's` position.
2. **5s** later after `unitDiedEvent` starts processing, remove the `unit` and the fire shape rendered when `unitDamagedEvent`.

### SoftBlockDestroyedEvent

`softBlockDestroyedEvent` should at least contain `softBlockId`. Missing one of them, or invalid values will make the game unable to proceed. Flag them in errorMessage if any.

Validate if:
- `softBlock` with `softBlockDestroyedEvent.softBlockId` exists in `gameState.softBlocks`.

`softBlockDestroyedEvent` does carry a `position` field on the wire, but it is intentionally ignored — position is always resolved via `softBlockGraphicsById`, for consistency with `unitDamagedEvent`/`unitDiedEvent`, which carry no `position` at all.

For each `softBlockDestroyedEvent`:

1. Look up the `softBlock's` position.
2. Render a fire shape on top of the blast and `softBlock`, representing the `softblock` is burning. Size is **42px** (larger than the 32px blast beam, so it visibly overlaps). Opacity is **70%**.
3. **5s** later, remove the `softBlock` and the fire shape.

### Rendering Sequence

This is the trickiest part. `Bombs` explodes with chain-reaction, meaning they could be exploded even if `countdown > 0`. Although backend has already sorted `gameEvents` chronologically, it doesn't explicitly state which `bombs` are exploded due to chain-reaction. An `Occupant` shouldn't be affected until the blast animation reaches its own `tile`.

```text
bombCountdownUpdatedEvent-1; bombId-001, countdown-0
bombCountdownUpdatedEvent-2; bombId-002, countdown-0
bombCountdownUpdatedEvent-3; bombId-003, countdown-2
bombExplodedEvent-4; bombId-001
bombExplodedEvent-5; bombId-002
bombExplodedEvent-6; bombId-003
unitDamagedEvent-7; unitId-007
unitDiedEvent-8; unitId-007
softBlockDestroyedEvent-9; softBlockId-009
unitDamagedEvent-10; unitId-010
unitDiedEvent-11; unitId-010
```

In the above example of `gameEvents` returned.

1. `bombCountdownUpdatedEvent-1`, `bombCountdownUpdatedEvent-2`, `bombCountdownUpdatedEvent-3` should be rendered concurrently first as they are the 1st group of `gameEvents`.
2. `bombExplodedEvent-4` and `bombExplodedEvent-5` should be rendered concurrently as their `countdown` reach **0**.
3. `bombExplodedEvent-6` should be rendered only when the blast reaches `bombId-003`.
4. `unitDamagedEvent-7`, `softBlockDestroyedEvent-9` and `unitDamagedEvent-10` should be rendered only when the blast reaches the corresponding `Occupant`.
5. `unitDiedEvent-8` and `unitDiedEvent-11` should be rendered after each of `unitDamagedEvent-7` and `unitDamagedEvent-10` handled. 

No backend enhancement (e.g. `bombChainReactedEvent`) is needed — chain reaction is derivable client-side. Bomb/unit/softBlock positions used for this derivation are sourced from the `gameState` snapshot taken immediately before `resolveTurn()` was called (not from the rendered `Graphics` objects, which are drawn with absolute pixel coordinates baked in and carry no readable tile position) — this is safe because no `unitMoved`/`bombPlaced` events occur within a `resolveTurn()` batch, so positions are static throughout.

For `bombExplodedEvent-N`: if `bombId-N`'s position appears in one or more **earlier** `bombExplodedEvent`'s `affectedPositions` within the same batch, it is a chain reaction. If multiple earlier events qualify, the causer is whichever yields the **smallest resulting delay** — `causer.offset + reachTime(causer.position, thisBomb.position)` — not necessarily the earliest event in array order. Delay this bomb's render by that computed offset from the causer's blast-start. Otherwise (no earlier match), render it immediately in the concurrent "countdown reached 0" group.

`unitDamagedEvent`/`unitDiedEvent`/`softBlockDestroyedEvent` follow the same rule: delay by `reachTime(causingBomb.position, occupant.position)` from the causing bomb's blast-start, using the same smallest-resulting-delay tie-break when more than one earlier `bombExplodedEvent`'s `affectedPositions` includes the occupant's position.

## Refresh Final Sanity Check

After all `gameEvents` are handled, call `getMatchState()` once for a final sanity check. Compare the freshly-fetched `gameState`'s `units`, `bombs`, and `softBlocks` against the client's own post-render bookkeeping (`unitGraphicsById`/`bombGraphicsById`/`softBlockGraphicsById`): each occupant's existence must match by ID (no occupant the client still renders but the server no longer lists, and vice versa). Unlike the turn-command sanity check (which diffs `Tile[][]` directly and so can compare position too), this check does not compare `position` — the bookkeeping maps only hold Graphics/Text objects, which don't retain a readable tile position, so re-deriving it here isn't worth the cost. If a mismatch is found, flag it in errorMessage, then re-render the `tiles`, `units`, `bombs` and `softBlocks` using the new `gameState`.

Replace the frontend stored `gameState` by the latest obtained one.

---

## Acceptance Criteria

1. Given `TurnPanel` is rendered, when `gameState.activeTeam` changes, then the header fill color updates to match `TEAM_COLORS[activeTeam]`.
2. Given `gameState.turn` > `gameCfg.maxTurns`, when `TurnPanel` renders, then the turn number renders in `0xff0000`.
3. Given a `bombCountdownUpdated` event with `countdown > 0`, when rendered, then the bomb's displayed number updates to the new countdown.
4. Given a `bombCountdownUpdated` event with `countdown === 0`, when rendered, then the bomb shows a red `!` instead of a number.
5. Given a `bombExploded` event, when rendered, then affected tiles show the blast effect and the bomb sprite is removed from the grid.
6. Given a `bombExploded` event whose `affectedPositions` includes an occupied tile, when rendered, then a fire shape renders on top of that occupant.
7. Given a `unitDamaged` event, when rendered, then a fire shape appears over the `unit`; if `newHp > 0` (unit survives), the fire is removed after 5s with no further event.
8. Given a `unitDamaged` event followed by a `unitDied` event for the same unit, when the `unitDied` event is processed, then 5s later the `unit` and its fire shape are both removed.
9. Given a `softBlockDestroyed` event, when rendered, then a fire appears over the `softBlock`; after 5s the `softBlock` is removed.
10. Given a gameEvent missing a required field or referencing a non-existent bombId/unitId/softBlockId, when encountered, then an error message is shown and rendering halts per "game unable to proceed."
11. Given a `bombExploded` event whose `bombId`'s position is not in any earlier `bombExploded` event's `affectedPositions` in the same batch, when rendered, then it renders immediately, concurrent with other "countdown reached 0" bombs.
12. Given a `bombExploded` event whose `bombId`'s position is in one or more earlier `bombExploded` events' `affectedPositions`, when rendered, then its render is delayed by `reachTime(causer.position, thisBomb.position)` from the causer's blast-start, where the causer is whichever earlier event yields the smallest resulting delay.
13. Given a `unitDamaged`/`unitDied`/`softBlockDestroyed` event caused by a bomb's blast, when rendered, then it is delayed by `reachTime(causingBomb.position, occupant.position)` from that bomb's blast-start, with both positions resolved via the pre-resolve `gameState` snapshot, not the event's own `position` field.
14. Given a bomb's blast is still in its lingering `BLAST_DURATION_MS` tail, when an unrelated gameEvent occurs, then that unrelated event renders concurrently, unblocked by the lingering blast.