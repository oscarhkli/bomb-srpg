# Roadmap

## Phase 1: Core Engine & Terminal Driver

- **Goal:** Implement the absolute minimum game rules running entirely in a local command line interface.
- **Scope:** Max 16x16 matrix, player placement, basic move/attack actions, and turn switching.
- **DoD:** A full match can be played out and won by text inputs via the terminal without a crash.

## Phase 2: Web Server & Transition from Terminal Driver

- **Goal:** Drop the console interface and build a graphical browser client.
- **Scope:** Core engine is wrapped in a Go HTTP server. The frontend renders the grid.
- **DoD:** Two local players can play a full pass-and-play match on a single browser window using standard HTTP requests.

## Phase 3a: Add WebSockets

- **Goal:** Upgrade the networking layer to support live, real-time online multiplayer between separate machines.
- **Scope:** Connection pools in Go, state synchronization broadcasting, and client disconnect handling.
- **DoD:** Two players on completely separate computers/browsers can join a unique game room via a URL and play a full match with real-time UI updates.

## Phase 3b: More Character Classes & Skills

- **Goal:** Expand game depth by transitioning from basic stats to a flexible, component-based unit and ability engine.
- **Scope:** Implement more unit types.
- **DoD:** New characters can be selected and use their unique skills inside the game, with both the web frontend rendering the visuals and the Go backend fully validating the custom actions.

## Phase 3c: Terrain & Power-Up Items

- **Goal:** Expand game depth by adding terrain effects and power-up items. The latter will dynamically alter the character's stats.
- **Scope:** Implement terrain effects. Code the stat-modifying triggers for Power-ups when stepped on.
- **DoD:** A character can move across varied terrain with accurate movement point deductions, and collect power-ups that modify backend stats.

## Phase 4: Add Computer Player with AI

- **Goal:** Introduce a single-player mode against an automated opponent.
- **Scope:** Heuristic-based enemy unit logic running inside a backend worker.
- **DoD:** A player can play a match against a local AI opponent that automatically calculates and executes its turns.

## Wish list
- Story Mode
- Replay