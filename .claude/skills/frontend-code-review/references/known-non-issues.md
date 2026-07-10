# Known non-issues

Patterns previously flagged by this skill that the user confirmed are not actually problems. Check the diff against this list before reporting anything similar — don't re-flag these unless the noted condition for revisiting has clearly changed.

## `console.log` of `roomId`/`playerTokens` in scene `create()`

**Applies to:** `MatchScene`/`DevBootScene`-era code during early frontend development, while there's no UI flow yet to display or inject match-join credentials for manual testing.

**Why not an issue (for now):** Deliberate temporary dev aid, not an oversight — `docs/frontend/match-p3-spec002.md`'s Goal explicitly requires logging these so they can be copied out of devtools for manual testing. It's a real client-side data-exposure pattern in the abstract (worth still mentioning as a one-line security note, not a ranked finding), and it's slated for removal once a concrete UI flow replaces manual devtool copy-paste.

**Revisit if:** a later diff adds a real match-join/auth UI flow that still logs tokens instead of routing them through that flow — at that point this stops being a temporary dev aid and becomes a real leak.

## Scheduled `scene.time.delayedCall`/`scene.tweens.add` calls with no explicit shutdown cleanup

**Applies to:** `web/src/rendering/resolveTurnPlayer.ts` (and any future code scheduling multi-second Phaser timers/tweens from a Scene).

**Why not an issue:** A Scene's `Clock` (`this.time`) and `TweenManager` (`this.tweens`) are both per-scene systems managed by the Scene's own lifecycle — per the installed `time-and-timers`/`scenes` Phaser 4 skill docs, the Clock is "managed by scene lifecycle events (`PRE_UPDATE`, `UPDATE`, `SHUTDOWN`, `DESTROY`)" and `scene.stop()` "clears display list and timers." Phaser already cancels pending `delayedCall`s and tweens when the owning scene shuts down — no manual `this.events.once('shutdown', ...)` cleanup is needed for timers/tweens specifically (as opposed to e.g. external event listeners or DOM references, which genuinely do need manual cleanup).

**Revisit if:** a future Phaser version changes this behavior, or code introduces a custom scene-transition path (e.g. reusing a live Scene instance across matches without a real `stop()`/`shutdown()`) that could bypass the normal lifecycle.
