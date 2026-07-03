# Phaser 4 idiom review

Phaser is a stateful, retained-mode game engine — unlike a typical web framework, nothing gets garbage-collected just because a component unmounted. Scenes, GameObjects, tweens, timers, and event listeners all live until something explicitly destroys them. Most Phaser bugs are lifecycle bugs: something created in `create()` outlives the scene, or something read in `update()` assumes state that hasn't been initialized yet.

**Don't re-derive Phaser API behavior from memory.** The installed Phaser 4 skills (`~/.claude/skills/<topic>/`) are the authoritative source for lifecycle, event, timer, and API details — consult the relevant one before asserting how something behaves. This file only adds the review-specific lens: what pattern in a diff counts as a finding, and how severe it is in *this* codebase.

## Where to look things up

| If the diff touches... | Consult skill | Watch for |
|---|---|---|
| `create`/`update`/`shutdown`, scene transitions, `this.scene.start/launch/switch` | `scenes` | Wrong transition method silently changing whether `shutdown` runs; state read in `update()` before `create()` finished |
| `.on(...)`/`.emit(...)`, custom events | `events-system` | Listener registered without a matching `.off`/teardown |
| `this.time.addEvent`, delayed calls | `time-and-timers` | Timer not cleared on scene stop/restart |
| `this.tweens.add` | `tweens` | Tween not killed on scene stop/restart |
| Anything importing a v3-only API or pattern | `v3-to-v4-migration` | Confirm it's not stale v3 usage — this repo is on 4.2.0 |
| New Phaser 4-only capability being used | `v4-new-features` | Confirm it's used correctly, not just because it's new |

## bomb-srpg-specific findings (not covered by the generic Phaser skills)

- **Async state races tied to the backend.** This app's scenes call the Go backend via `web/src/engine/api.ts`. If a scene starts a `fetch`/`await` and then touches GameObjects or reads `TrueState`/`WorkingState`-derived data afterward, check whether the scene could have been stopped or the state reset in the meantime — Phaser won't guard against touching a destroyed GameObject, it just throws or no-ops.
- **Grid↔pixel coordinate conversion.** Phaser's coordinate system is screen-space pixels, origin top-left; this repo's grid convention is `Grid[Y][X]` row-major, `(0,0)` top-left (root `CLAUDE.md`). A conversion bug between the two is an easy off-by-one and easy to miss — check any tile↔pixel math directly against the grid convention, not just "does it look plausible."
- **GameObject churn.** Creating new GameObjects every frame/state-update instead of reusing/repositioning existing ones works but degrades over a long session. Medium severity — correctness is usually fine, it's a waste-not-a-crash issue.
- **Magic numbers that belong in `constants.ts`.** This repo already centralizes pixel positions/depths/colors there; new magic numbers duplicating that pattern are a smell, not a bug — low severity.
