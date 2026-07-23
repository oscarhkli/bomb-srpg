# Visual Vocabulary

Named shorthand for UI patterns repeated across specs. Cite by name instead of re-describing values inline.

## Structural

All three role names below are just Phaser `Container`s — the name conveys the UI role.

- **Scene** — a Phaser Scene: own lifecycle, display list, camera, input plugin.
- **Container** — a Phaser Container: groups GameObjects under one shared transform, no own lifecycle.
- **Page** — a swappable content view within a scene body, shown one at a time (e.g. `UnitPage`, `StagePage`). The scene chrome (`region`s) stays put while the active `Page` changes.
- **region** — a fixed layout band of scene chrome, one per scene layout (e.g. `HeaderRegion`, `NavRegion`).
- **Panel** — a content area inside a `Page` body (e.g. `FormationPanel`, `ArchetypesPanel`).

## Shapes

- **rounded-square(size, fill)** — strictly w == h. e.g. `rounded-square(96px, 0x4c4c4c)`.
- **rounded-rect(w, h, fill)** — w != h. Kept distinct from rounded-square even though a square is technically a rectangle — pick whichever term matches the actual aspect ratio.
- **pill(w, h, fill)** — rounded-rect with fully rounded ends (corner radius = h/2).

## Buttons

- **PillButton(size)** — pill-shaped button, `PANEL_BUTTON_*`/`LIFECYCLE_BUTTON_*` style (constants.ts). `size` ∈ { panel, lifecycle, lifecycle-small }, matching the constants' named variants. Canonical shape source: TurnCommandPanel buttons (p3-spec003-match.md). A caller may keep the style (fill/border colors) while overriding the size with an explicit `pill(w, h)` — state the size when doing so.
- **DisabledButton** — same shape as its enabled counterpart, filled with `DISABLED_BUTTON_COLOR` (0x999999, constants.ts).
- **BackButton** — shared back-navigation button: `rounded-square(64px, 0x4c4c4c)` with a `⮐` glyph centered (36px, `0xffffff`). Distinct rendering from MatchScene's own in-match back control. Introduced by p3-spec009-stage.md. `⮐` (`U+2B90`) is a symbol, not emoji — no emoji fallback, and Roboto/system-sans coverage is spotty, so it's the one glyph here that can tofu (▯). **Backup:** fall back to the `Back` text label (`BACK_BUTTON_LABEL`, constants.ts) if manual testing shows tofu.
- **TeamBadge(w, h)** — rounded-rect/pill filled with `TeamColor(team)`, holding a `P{X}` label.

## Transitions

Two different things, easy to confuse:

- **fadeTransition** — an animated camera fade-out → fade-in over `FADE_MS` (200ms) each, via the Scene camera's `fadeOut`/`fadeIn`. Plays once to mask a Scene/`Page` swap; the viewport goes fully dark and back. The default scene / `Page` transition.
- **Dim(color, alpha)** — a static, persistent semi-transparent overlay drawn over content at a fixed alpha to de-emphasize it behind a modal. No default value — each spec states its own `color`/`alpha`. Reference instance: ConfirmDialog's `0x1a1a1a @ 60%`.

## Colors

- **TeamColor(team)** — shorthand for `TEAM_COLORS[team]` (constants.ts), with `TEAM_COLOR_FALLBACK` for unrecognized values.

## Typography

- Default text: 12px, `0xffffff`, unless a spec states otherwise.

## Sizing & Layout

Layout numbers should be **rooted, not scattered**. A magic number isn't "a literal" — it's an *unnamed literal that duplicates a fact defined elsewhere*.

- **root constant** — a named absolute px, defined once, that everything else derives from. The recursion of "relative to what?" bottoms out here. Roots: the canvas size, a base spacing unit, and the scene margin.
- **derived value** — any other size/position, expressed in terms of a root.

Rules when placing anything:

1. **Canvas size** comes from Phaser (`this.scale.width` / `this.scale.height`, = 1280×720), never a retyped `1280`/`720`.
2. **Positions** anchor to edges + margin or to a neighbor, not absolute coordinates: `this.scale.width - MARGIN - w`, or `prev.y + prev.h + GAP` — not `x = 1136`.
3. **Sizes** derive from available space or a base-unit multiple (`BASE * n`), not a frozen per-element constant. The board tile fills its region (`floor(regionH / rows)`), rather than a hardcoded `TILE_SIZE`.

**Litmus test:** *"If the canvas weren't 1280×720, would this number still be right?"* If no, it's a disguised duplicate of a root — name it and derive.

> Scope: this is the going-forward habit for new layout code. Existing fixed-px constants (`TILE_SIZE`, panel widths) are retrofitted later in the visual pass, not eagerly.
