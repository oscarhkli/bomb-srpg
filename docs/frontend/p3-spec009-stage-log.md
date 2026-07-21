---
title: "Log: p3-spec009-stage"
---

# Known Issues

Found while completing the in-progress implementation (test-fixing pass), not gaps in
`p3-spec009-stage.md` itself — logged here for traceability.

1. **`ArchetypesPanel` isn't vertically scrollable.** The spec's own wording frames this as a
   forward-looking need ("at the moment there should be 4 `UnitCards` a row... in future there
   will be more than 16 archetypes, so `ArchetypesPanel` should be vertically scrollable"), and
   the backend currently exposes only 3 selectable archetypes (Fighter, Witch, Bandit — see
   `engine/presets.go` `archetypesRegistry()`), which fit one row with no overflow. `UnitPage.ts`
   renders a static, non-scrolling grid; its click handlers read `this.slots` live so they don't
   go stale, but nothing clips or scrolls content that overflows the body region.
   **Status: Deferred.** Revisit once the archetype catalog grows enough to overflow a single row
   (or before then, if a UX pass wants scrolling proactively).
2. **`fireCameraFadeOutComplete()` test helper (`web/src/test/sceneHelpers.ts`) used `.find()`
   instead of the last match** when looking up the registered `'camerafadeoutcomplete'` `.once()`
   listener. A real `.once()` listener auto-removes itself after firing, but the mock's
   `mock.calls` retains every registration ever made — so a test driving more than one
   fadeTransition in sequence (e.g. UnitPage 2 -> UnitPage 1, or UnitPage 2 -> StagePage -> back)
   would re-fire the *oldest* transition's callback instead of the latest, silently re-rendering
   the wrong Page. Manifested as `MatchSettingsScene.test.ts`'s AC14/AC15 BackButton tests landing
   on the wrong Page. Fixed by taking the last matching call instead of the first.
   **Status: Solved.**
3. **`MatchSettingsScene.test.ts`'s `graphicsSince(since)` pattern broke when `since` was captured
   before `bootScene()` ran.** `BackButton` is drawn once, synchronously, in `create()` — before
   the async `getCatalog()` resolves and the first Page renders — so a `since` checkpoint taken
   before scene creation incorrectly counted the `BackButton`'s own `Graphics` as the Page's first
   object (an off-by-one that shifted every subsequent index: `UnitSlot`s, `UnitCard`s, and
   `NextButton` all resolved to the wrong `Graphics` instance). Fixed by adding a
   `currentPageGraphics()` helper (`lastGraphics(7 + N)`) that grabs the trailing N graphics a
   single `renderActivePage()` call is known to produce, independent of what was rendered earlier
   in the same test.
   **Status: Solved.**
