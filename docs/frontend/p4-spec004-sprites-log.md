# p4-spec004-sprites — Known Issues

## Halving STAGE_CARD_SELECTED_BORDER_WIDTH collides with PANEL_BUTTON_BORDER_WIDTH

**Status:** Solved

The spec's general rule halves every absolute layout constant in `MatchSettingsScene`. `STAGE_CARD_SELECTED_BORDER_WIDTH` (4px) halves to 2px under that rule, which lands exactly on `PANEL_BUTTON_BORDER_WIDTH` (2px) — the shared, unhalved border width every other panel/card uses. A selected `StageCard` would render with the same border width as an unselected one, losing the visual distinction the border exists to convey.

**Remark:** Not a spec-wording problem — the halving rule is correct for nearly every constant it touches, this is a one-off collision between a halved value and an unrelated shared constant. Resolved by rounding up to 3px instead of the literal half, confirmed with the user, keeping the selected border visually distinct from the 2px default.

## Scene Exit's shutdown camera removal was dead code

**Status:** Solved

The spec originally required removing `MatchSettingsScene`'s camera on `shutdown`, mirroring `MatchScene`'s pattern. Found during code review: `Phaser.Cameras.Scene2D.CameraManager` already registers its own `shutdown` listener when the scene starts (before `create()` runs), which destroys every camera on that scene the moment `shutdown` fires. Registration order means that listener always runs before ours, so the explicit `this.cameras.remove(settingsCamera)` call never had anything left to remove — a no-op, not a leak.

**Remark:** Removed the dead line and corrected the spec's Scene Exit section to state Phaser's own teardown is sufficient. No behavior change — cameras were already being cleaned up correctly by Phaser regardless of this code.

## STAGE_CARD_NAME_FONT_SIZE's flat -4px reduction overflowed the halved StageCard

**Status:** Solved

The general font-sizing rule (flat -4px, not a percentage) left `STAGE_CARD_NAME_FONT_SIZE` at 32px while `STAGE_CARD_SIZE` halved from 160px to 80px — an 89%-of-original font inside a 50%-of-original container. Longer preset names ("Standard", "Divided") overflowed and visually overlapped adjacent StageCards.

**Remark:** Unlike chrome text anchored to fixed-height regions, this label's container (`STAGE_CARD_SIZE`) is itself a halved absolute constant, so the label needs to scale with it. Set `STAGE_CARD_NAME_FONT_SIZE` to 16px (matching the card's own halving) instead of the flat rule, and enabled `useAdvancedWrap` on the name text as a safety net for future longer preset names.
