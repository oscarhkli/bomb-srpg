---
title: "Phase 4.10: Swapping Sprites for Existing Vector Graphics in MatchSummaryPanel"
---

# Phase 4.10: Swapping Sprites for Existing Vector Graphics in MatchSummaryPanel

**Skip everything below - copy-and-paste only.**

## Context

This spec replaces the vector graphic of the buttons in `TurnCommandPanel` and `ConfirmDialog` in MatchScene by Pixel Art sprites. Also, it adjusts the layout according to the sprites.

> **Shared vocabulary:** This spec relies on shared terms and design conventions — `Page`, `region`, `Panel`, `fadeTransition`, `BackButton`, `TeamBadge`, the `render*`/`draw*` split, etc. — defined in [`VISUAL_VOCAB.md`](./VISUAL_VOCAB.md). Read it first.

## Goal

- `MatchScene` displays pixel art assets of buttons in `TurnCommandPanel` and `ConfirmDialog`.
- Add visual effects to the swapped buttons.

## Non-Goal

- Complete Pixel Art adoption for `TurnCommandPanel` and `ConfirmDialog`

## Scene Entry

No change from spec005.

---

## Sprites

Sprites are exported **trimmed** — the delivered art's trim is not guaranteed to be centered within the untrimmed source canvas (`sourceSize`/`spriteSourceSize`), so origin cannot assume a symmetric crop.

| Entity            | Texture Key       | Path (relative to `sprites/`)         | Type             | Remarks                           |
| ----------------- | ----------------- | ------------------------------------- | ---------------- | --------------------------------- |
| Button (Neutral)  | button_neutral    | buttons/Button-Neutral.png (+ .json)  | atlas (aseprite) |                                   |
| Button (Selected) | button_selected   | buttons/Button-Selected.png (+ .json) | atlas (aseprite) | 2px taller and wider on each side |
| Button (Clicked)  | button_clicked    | buttons/Button-Clicked.png (+ .json)  | atlas (aseprite) |                                   |
| Label (Move)      | button_label_move | buttons/Button-Move.png (+ .json)     | atlas (aseprite) |                                   |
| Label (Bomb)      | button_label_bomb | buttons/Button-Bomb.png (+ .json)     | atlas (aseprite) |                                   |
| Label (Back)      | button_label_back | buttons/Button-Back.png (+ .json)     | atlas (aseprite) |                                   |
| Label (Yes)       | button_label_yes  | buttons/Button-Yes.png (+ .json)      | atlas (aseprite) |                                   |
| Label (No)        | button_label_no   | buttons/Button-No.png (+ .json)       | atlas (aseprite) |                                   |

Buttons are all composite component, meaning Button and Label are stacked using a container to form a component, with Label has a slight higher depth. The Container is the button's interactive hit-area — Button and Label themselves are not individually interactive.

### Alignment

Button and Label are trimmed from the same Aseprite source canvas. Both GameObjects are added to the Container at local `(0, 0)` with `origin (0.5, 0.5)`. Phaser positions a trimmed atlas frame using its trim offset (`spriteSourceSize`/`sourceSize`) as if the full untrimmed canvas were still present, so Label renders exactly where it was hand-drawn relative to Button — no manual per-label offset is needed, and this holds across every Button state (Neutral/Selected/Clicked) despite their differing trimmed sizes.

The Clicked state's 2px downward Label shift (see below) is an explicit runtime offset layered on top of this baseline, not part of the trim alignment itself.

### Visual Effects

In addition to the click handler in the vector graphic version, more visual handling will be added.

### Neutral

Used when the button is in neutral state. The frontend should render `button_neutral`.

### Disabled

Used when the button is in disabled state (same as the current conditions used in vector graphic version). It shares the sprites as `Neutral`, with `colorMatrix.desaturate()` for the whole container so that button and label are both in its greyscale color.

### Selected

Used when the button is not in disabled state and User is selecting / mouseover the button. The frontend should render `button_selected`.

### Clicked

Used when the button is not in disabled state and User clicks / pressed / mousedown the button. The frontend should render `button_clicked`. Note that to align with the button, label should shift **2px** downwards when entering this state, and return to the original position when the button is not clicked.

> Future work: once a second button family (not using this sprite set) needs the same Neutral/Disabled/Selected/Clicked state machine, extract it into `VISUAL_VOCAB.md` as a named pattern (texture-swap rules, the 2px Clicked label-shift, desaturate-on-disabled) instead of restating it per spec. Not needed yet — `TurnCommandPanel`/`ConfirmDialog` are the only adopters so far.

## Visual Spec for TurnCommand Panel

- `TurnCommandPanel` should now be 100% width. Height is not defined - letting the Buttons the expand its height.
- `TurnCommandPanel` should stay **40px** above the edge of the canvas.
- No more padding on the 4 sides of `TurnCommandPanel`.
- Instead of placing in 2x2, all buttons inside should be stacked 1 per row, center aligned.
  - Move
  - Bomb
  - Back
- Each button inside should leave a `4px` vertical space separated — enough that a hovered button's Selected size (2px taller on each side) never overlaps its neighbor.

## Visual Spec for ConfirmDialog

- Instead of placing in 1 row, 2 buttons inside should be stacked 1 per row, center aligned.
  - Yes
  - No
- Each button inside should leave a `4px` vertical space separated — enough that a hovered button's Selected size (2px taller on each side) never overlaps its neighbor.
- `NoButton` should be placed at the bottom of `ConfirmDialog`.

---

## Acceptance Criteria

1. Given `MatchScene` has loaded a match, when the unit is clicked, then all buttons in `TurnCommandPanel` display their pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
2. Given a button in `TurnCommandPanel`, when the button is disabled due to some reasons, then the button with the label should turn in its greyscale color.
3. Given a button in `TurnCommandPanel`, when the button is not disabled and selected, then the button with the label should change the sprite to mimic glow effect.
4. Given a button in `TurnCommandPanel`, when the button is not disabled and clicked, then the button with the label should change the sprite to mimic click effect, and label should shift **2px** downwards.
5. Given `MatchScene` has loaded a match, when `ConfirmDialog` pops up, then all buttons in `ConfirmDialog` display their pixel-art sprite textures instead of the Phase 3 vector-graphics placeholders.
6. Given a button in `ConfirmDialog`, when the button is not disabled and selected, then the button with the label should change the sprite to mimic glow effect.
7. Given a button in `ConfirmDialog`, when the button is not disabled and clicked, then the button with the label should change the sprite to mimic click effect, and label should shift **2px** downwards.

## Log

Implementation issues found during the build (non spec gaps) are tracked in [p4-spec006-sprites-log.md](./p4-spec006-sprites-log.md).
