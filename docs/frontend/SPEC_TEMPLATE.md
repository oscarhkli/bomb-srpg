# Scene Spec Template

Copy this skeleton for each new scene spec. Remove optional sections that don't apply.

---
title: "Phase X.Y: [Spec Title]"
---

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

| Field | Type | Source |
|---|---|---|
| `field` | `type` | Where it comes from |

### Initialisation sequence

Steps `create()` must perform, in order.

---

<!-- OPTIONAL SECTIONS — remove if not applicable -->

## Layout *(optional — scenes with a game world)*

Camera model, canvas resolution, coordinate system.

## Data Fetching *(optional — scenes that call the backend)*

Which API functions are called, when, and how often.

## Visual Spec *(optional — scenes with custom rendering)*

What each rendered element looks like (shape, color, size).
Reference `constants.ts` for named values; avoid hardcoded hex here.

## Scene Exit *(optional — scenes with multiple destinations)*

| Trigger | Destination |
|---|---|
| Event or action | Next scene |

## Dev Bootstrap *(optional — prerequisite scene not yet built)*

Temporary scaffolding to run this scene in isolation during development.
Remove once the real predecessor scene is implemented.

---

## Acceptance Criteria

1. Given … When … Then …
2. Given … When … Then …
