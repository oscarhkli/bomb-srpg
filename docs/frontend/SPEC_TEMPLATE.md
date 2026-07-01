# Scene Spec Template

Copy this skeleton for each new scene spec. Remove optional sections that don't apply.

---

## title: "Phase X.Y: [Spec Title]"

---

# [Scene name and one-line purpose]

## Context

Why this scene exists. What state the game is in when the player reaches it.

## Goal

- Bullet list of what this scene must do.

## Non-Goal

- What is explicitly deferred to a later spec.

## Scene Entry

Who launches this scene and what data it receives.

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
