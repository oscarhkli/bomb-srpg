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
- [ ] CLI terminal
  - [ ] CLI Controller
  - [ ] Display
  - [ ] GameEvent
  - [ ] Command
  - [ ] TurnCommand;
- [ ] Starting turns
  - [ ] Environment setup
  - [ ] Sudden death

## Phase 2: Web Server & Transition from Terminal Driver
- **Goal:** Drop the console interface and build a graphical browser client.
- **Scope:** Core engine is wrapped in a Go `net/http` server reading JSON payloads. The frontend renders the grid using zero-dependency Vanilla JavaScript and an HTML5 Canvas layer. Includes Optimistic UI client rendering.
- **DoD:** Two local human players can play a full pass-and-play match on a single browser window using standard HTTP requests. Terminal runner is deprecated or isolated.

### TODO
- [ ] Web Server & HTTP request migration
- [ ] Frontend display
- [ ] Frontend navigation
- [ ] Deployment

## Phase 3a: Add WebSockets (Optional Branch A)
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

## Phase 3b: More Character Classes & Skills (Optional Branch B)
- **Goal:** Expand game depth by transitioning from basic stats to a flexible, component-based unit and ability engine.
- **Scope:** Implement advanced unit types (Archers with min/max range limits, Mages utilizing Area-of-Effect parameters, Flying units overriding structural obstructions).
- **DoD:** New characters can be selected and use their unique skills inside the game, with both the web frontend rendering the visuals and the Go backend fully validating the custom actions.

### TODO
- [ ] Skill, e.g., prolonging the count down
- [ ] Advance path finding algorithm (e.g., float, jump, etc.)

## Phase 3c: Terrain & Power-Up Items (Optional Branch C)
- **Goal:** Expand game depth by adding reactive terrain effects and power-up items. The latter will dynamically alter the character's stats during turn validation.
- **Scope:** Power-up spawn math, movement resolution interceptors, and dynamic terrain modifiers (mud slowing navigation, lava shortern bomb countdown, water extinguishing explosives).
- **DoD:** A character can move across varied terrain with accurate movement point deductions, roll back gathered power-ups correctly on turn reset, and permanently collect buffs that modify backend stats upon commitment.

### TODO
- [ ] Terrains: Lava, Water
- [ ] Softblock with / without items

## Phase 4: Add Computer Player with AI
- **Goal:** Introduce a single-player mode against an automated opponent.
- **Scope:** Heuristic-based enemy unit logic running inside an asynchronous backend goroutine worker.
- **DoD:** A player can play a match against a local AI opponent that automatically calculates and executes its movements when its turn segment activates.

### TODO
- [ ] Risk management???

## Wish list
- Story Mode (pre-req: Computer Player)
- Replay (pre-req: Web server)