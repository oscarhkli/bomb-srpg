# Known non-issues

Patterns previously flagged by this skill that the user confirmed are not actually problems. Check the diff against this list before reporting anything similar — don't re-flag these unless the noted condition for revisiting has clearly changed.

## `console.log` of `roomId`/`playerTokens` in scene `create()`

**Applies to:** `MatchScene`/`DevBootScene`-era code during early frontend development, while there's no UI flow yet to display or inject match-join credentials for manual testing.

**Why not an issue (for now):** Deliberate temporary dev aid, not an oversight — `docs/frontend/match-p3-spec002.md`'s Goal explicitly requires logging these so they can be copied out of devtools for manual testing. It's a real client-side data-exposure pattern in the abstract (worth still mentioning as a one-line security note, not a ranked finding), and it's slated for removal once a concrete UI flow replaces manual devtool copy-paste.

**Revisit if:** a later diff adds a real match-join/auth UI flow that still logs tokens instead of routing them through that flow — at that point this stops being a temporary dev aid and becomes a real leak.
