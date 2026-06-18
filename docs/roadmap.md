# Project Roadmap

## Phase 1: Core Engine & Terminal Driver
- **Goal:** Implement the absolute minimum game rules running entirely in a local command line interface.
- **Scope:** Bounded max 16x16 matrix (e.g., 7x7 dynamic support), player placement, 2 basic unit types (King, Soldier), adjacent move/attack actions, basic command line parsing, and turn switching.
- **DoD:** A full match can be played out and won by text inputs via the terminal without an application crash.

### TODO
- [x] Basic models & presets Archetype and Stages
- [x] New game
- [x] Path finding algorithm (for movement, bomb range, stage sanity check, etc.)
- [x] Sandbox transaction handling (commit/reset)
- [x] Match action management
  - [x] Unit movement
  - [x] Bomb placement
- [x] Resolving turns
  - [x] Bomb ticking
  - [x] Bomb detonation, chain reaction
  - [x] Casualty updates
  - [x] Result calculation
- [x] Starting turns
  - [x] Environment setup
  - [x] Sudden death
- [x] CLI terminal
  - [x] CLI Controller
  - [x] Start/Restart Game
  - [x] Display
  - [x] GameEvent
  - [x] Command
  - [x] Surrender
  - [x] TurnCommand
- [x] Victory Condition
- [x] In-turn movement restriction

## Phase 2: Web Server & Headless API Implementation
- **Goal:** Setup web server embedding the game engine so a match can be played from start to finish purely via HTTP requests.
- **Scope:** Core engine is wrapped in a Go `net/http` server reading JSON payloads.
- **DoD:** Two local human players can play a full pass-and-play match using standard HTTP requests.

### TODO
- [x] Init router Setup 
- [ ] HTTP Handlers for Game Setup
  - [x] List Archetypes
  - [x] New Match with user-defined Game Config
- [ ] HTTP Handlers for Turn resolution and Match lifecycle
  - [ ] Reset
  - [ ] Commit
  - [ ] Surrender
- [ ] HTTP Handlers for TurnCommands
  - [ ] Move
  - [ ] PlaceBomb
- [ ] Match Room
 - [x] Creation
 - [ ] Housekeep based on the last activity time
 - [ ] Navigation with Match, Game cycle
- [ ] Deployment

## Phase 3: Graphical Browser Client & UI Integration
- **Goal:** Drop the console interface and build a graphical browser client.
- **Scope:** The frontend renders the grid using Phaser.js and an HTML5 Canvas layer. Includes Optimistic UI client rendering.
- **DoD:** Two local human players can play a full pass-and-play match on a single browser window using standard HTTP requests. Terminal runner is deprecated or isolated.

### TODO
- [ ] Phaser.js Engine & Asset Loading Boilerplate
- [ ] Match Lounge
- [ ] Board & Sprite Rendering from JSON State
- [ ] Input Mapping (Converting Clicks to HTTP Commands)
- [ ] Action Log Animation Playback
- [ ] Local Terminal Code Deprecation/Isolation
- [ ] Deployment

## Phase 4a: Add WebSockets
- **Goal:** Upgrade the networking layer to support live, real-time online multiplayer between separate machines.
- **Scope:** Connection pool management in Go, game room/lobby routers, and client disconnect handling.
- **DoD:** Two players on completely separate computers/browsers can join a unique game room via a URL and play a full match with real-time UI synchronization without manual page refreshes.

### TODO
- [ ] WebSockets setup
- [ ] Room
- [ ] Multiplayer management
  - [ ] Join/leave room
  - [ ] Room admin
  - [ ] Interruption handling

## Phase 4b: More Character Classes & Skills
- **Goal:** Expand game depth by transitioning from basic stats to a flexible, component-based unit and ability engine.
- **Scope:** Implement advanced unit types (Archers with min/max range limits, Mages utilizing Area-of-Effect parameters, Flying units overriding structural obstructions).
- **DoD:** New characters can be selected and use their unique skills inside the game, with both the web frontend rendering the visuals and the Go backend fully validating the custom actions.

### TODO
- [ ] Skill, e.g., prolonging the count down
- [ ] Advance path finding algorithm (e.g., float, jump, etc.)

## Phase 4c: Terrain & Power-Up Items
- **Goal:** Expand game depth by adding reactive terrain effects and power-up items. The latter will dynamically alter the character's stats during turn validation.
- **Scope:** Power-up spawn math, movement resolution interceptors, and dynamic terrain modifiers (mud slowing navigation, lava shortern bomb countdown, water extinguishing explosives).
- **DoD:** A character can move across varied terrain with accurate movement point deductions, roll back gathered power-ups correctly on turn reset, and permanently collect buffs that modify backend stats upon commitment.

### TODO
- [ ] Terrains: Lava, Water
- [ ] Softblock with / without items

## Phase 5: Add Computer Player with AI
- **Goal:** Introduce a single-player mode against an automated opponent.
- **Scope:** Heuristic-based enemy unit logic running inside an asynchronous backend goroutine worker.
- **DoD:** A player can play a match against a local AI opponent that automatically calculates and executes its movements when its turn segment activates.

### TODO
- [ ] Risk management???

## Wish list
- Story Mode (pre-req: Computer Player)
- Replay (pre-req: Web server)