---
title: "Phase 3.6: Victory Cutscene"
---

# Victory Cutscene

## Context

Phase 3.5 implements the full game cycle of a match, but when the match concludes, the frontend didn't stop, resulting an error msg like `match already ended`. This spec adds a `VictoryCutscene` to conclude the match, and provide a way to restart the match.

## Goal

- Render `VictoryCutscene` when the match ends.
- Add `rematchButton` to restart a match.

## Non-Goal

- Polished animations or tweens (easing curves, squash/stretch, particle effects, etc.)
- HUD / status panel.
- Detailed implementation of `MatchSetupScene` - a rough page for scene entry is accecptable. Detailed part will be initiated in `stage-p3-spec001.md`

## Scene Entry

No change from spec001.

### Data on arrival

| Field   | Type   | Source              |
| ------- | ------ | ------------------- |
| `field` | `type` | Where it comes from |

### Initialisation sequence

Steps `create()` must perform, in order.

---

<!-- OPTIONAL SECTIONS — remove if not applicable -->

## Layout _(optional — scenes with a game world)_

Camera model, canvas resolution, coordinate system.

## Data Fetching _(optional — scenes that call the backend)_

Which API functions are called, when, and how often.

## Visual Spec _(optional — scenes with custom rendering)_

What each rendered element looks like (shape, color, size).
Reference `constants.ts` for named values; avoid hardcoded hex here.

## Scene Exit _(optional — scenes with multiple destinations)_

| Trigger         | Destination |
| --------------- | ----------- |
| Event or action | Next scene  |

## Dev Bootstrap _(optional — prerequisite scene not yet built)_

Temporary scaffolding to run this scene in isolation during development.
Remove once the real predecessor scene is implemented.

---

## Acceptance Criteria

1. Given … When … Then …
2. Given … When … Then …

## Log _(optional - remove it if no implementatioun issue is found)_

Implementation issues found during the build (non spec gaps) are tracked in [`match-p3-spec001-log.md`](./match-p3-spec001-log.md).
