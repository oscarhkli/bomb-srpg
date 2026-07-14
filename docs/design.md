# Technical Design Specification

## 1. Architectural Principles

- **Authoritative Server / Zero Trust**: The backend (`engine`) acts as the single source of truth. Incoming user command payloads are treated as unverified requests. The engine recalculates all pathfinding and rules independently before mutating memory.
  - _Zero-Trust Parameters_: Property deductions—such as initial bomb countdowns based on terrain or player skills—are processed authoritatively on the server boundary. Client payloads cannot dictate property states.
- **State Sandboxing Pattern**: At the start of a turn, the engine clones the definitive `GameState` into a temporary `WorkingState` sandbox. All mid-turn actions (moving, placing bombs, picking up power-ups) alter this sandbox layer.
  - A `/reset` or `ResetTurn()` command discards the sandbox and re-clones the definitive checkpoint (`m.TrueState`), wiping any uncommitted player actions.
  - A `/commit` or `ResolveTurn()` command executes batch-resolution loops against the sandbox, advances the timeline, and permanently promotes the sandboxed state to the master history via a deep-copy clone.

* **Batch Resolution**: All damage calculations, terrain updates, bomb explosions, and chain reactions are deferred until a turn is committed. The resolution engine runs synchronously and returns a chronological `Animation Event Queue` string/binary array payload to the client.

- **Synchronous Execution Bound**: Given the low-density metrics (16x16 maximum matrix, <= 10 active characters), all grid lookups, BFS pathfinding, and explosion cascade evaluations are run synchronously to avoid multi-threaded race conditions.

## 2. Spatial Mapping & State Model Definitions

- **Stage Dimensions**: Dynamic N x M grid layout (supporting ranges from 7x7 up to a maximum bound of 16x16).
- **Storage Type**: Rows are allocated via dynamic Go slices (`[][]Tile`) to support custom asymmetrical maps without recompiling code.
- **Coordinate Mapping**: Layout is indexed as `Grid[Y][X]`. The top-left corner of the map is designated as `(0,0)`.

* **Dynamic Boundary Rule**: Every check evaluates dynamically against active bounds: `0 <= X < len(grid)` and `0 <= Y < len(grid)`.

- **Memory Normalization**: The Board Matrix tracks tile references via minimal structural fields (`OccupantType`, `OccupantID`). The GameState Engine maintains the master map directory of active entities. Moving an object updates only the cell metadata, leaving base entity metrics untouched.

## 3. Strongly Typed Identity Contracts & Bitmask Configurations

- **Compile-Time Contract Safety**: To guarantee type safety and eliminate argument-swapping bugs, core entity lookups are wrapped in custom, power-of-2 byte-aligned semantic primitives.
- **UnitID (uint8)**: Split internally into two equal 4-bit nibbles via `(TeamID << 4) | PlayerIndex`. This accommodates up to 15 unique teams and 15 players per team, mapping perfectly onto a single hexadecimal byte for clear log readability (e.g., `0x23` means Team 2, Player 3).
- **BombID (uint32)**: Packed to represent absolute timeline metadata using clear byte boundaries: `(UnitID << 24) | (CurrentTurn << 16) | BombCounterShift`.
- **TurnBombCounter Invariant**: A state-bound tracking sequence index that advances on the active sandbox frame. Because it is bound directly to the sandbox container, it resets on a turn-reset request, preventing ID collisions or frontend asset desynchronization.
- **SystemUnitID Contract**: The identifier `UnitID(0)` (Team 0, Player 0) is explicitly reserved as the authoritative environmental actor. It is utilized to log and process automated environmental events, such as sudden-death hazards, without creating phantom player models.

## 4. Pathfinding, Trajectory Vectors & Collision Engine

- **Transient Snapshot Pattern**: Instead of heavy OOP decorator patterns or permanent state mutations, movement capabilities are passed into pathfinding as a short-lived, discardable `MovementRule` context instruction slip generated on the fly using active character statistics.
- **High-Performance Bitmask Matrix**: Entity pass-through permissions are packed into a highly optimized, 1-byte bitmask field (`PassFlags` uint8). This completely eliminates nested slice iteration loops inside the pathfinder, collapsing obstacle evaluation down to a bitwise operation.
- **Separation of Concerns (Reachability vs. Legality)**:
  - The pathfinder function (`FindReachableTiles`) has exactly one job: mapping spatial reachability and step distance tracking (`map[Coordinate]int`), treating the origin tile as step 0.
  - Target landing restrictions (e.g., landing on an item tile or targeting allies) are business rules evaluated independently by respective **Action Handlers** _after_ pathfinding returns.
- **Unified Impact Absorption Crucible**: Straight-line calculation rays (character walking, bomb explosions) and corner-wrapping paths (sanity check). An explicit impact flag (`StopOnFirstNonUnitOccupant`) forces the ray to register a valid hit on a solid obstacle (SoftBlock, Bomb, Item) but instantly terminates the vector to shield cells behind it.
- **Snapshot-Based Detonation**: All cascading bomb explosions within an intra-turn phase resolve at the exact same physical millisecond. To prevent ray-truncation anomalies, the engine queries a read-only 2D snapshot copy of the board captured at the start of the resolution pass (`cloneGridSnapshot()`). Destructible soft blocks flag their destruction inside delayed registries but remain solid, ray-blocking obstacles until the end of the pass to ensure perfect unit shielding.
- **Intra-Turn Damage Capping**: Units caught in overlapping blast patterns or multiple cascading explosions lose a flat maximum of exactly 1 HP, matching the low-density 1-HP character pacing rules. Damage calculations utilize a local boolean presence set to record injuries rather than an accumulator, preventing health overflow. All entity modifications, soft wall dissolutions, and tile clear passes execute as a simultaneous batch flush at the very end of the loop.

## 5. Online Multiplayer Synchronisation & State Machine Loops

- **Turn Secrecy Pattern**: Active turn planning is fully hidden from the opposing player to preserve the Turn Reset capability. Opponents see a passive waiting status during the planning phase.
- **Unified Event Broadcast**: Upon turn commitment, the backend generates an identical chronological Action Queue array and distributes it to both clients.
- **Frontend Queue Playback**: Clients process incoming batch payloads using a sequential async loop, ensuring both players watch animations unfold with perfect deterministic lockstep alignment.
- **Explicit Turn Start Flow**: The server does NOT auto-call `StartTurn()` on match creation. Instead, the client explicitly calls `POST /match/start-turn` to begin each turn (including Turn 1). This allows the UI to render the clean initial state, then animate sudden-death bomb drops when they occur on Turn 1 with `MaxTurns=0`.
- **Turn Startup Sudden Death Checks**: State machine transition and boundary rules—such as checking if `TrueState.Turn >= Config.MaxTurn`—are evaluated at the very beginning of a new turn (`StartTurn()`). This ensures map alterations and automated sudden-death bomb injections are fully populated and rendered before a player can input commands. Setting `MaxTurn = 0` forces instant sudden death on Turn 1.
- **Decoupled Analytical Queries**: Victory evaluations (`EvaluateVictoryConditions()`) are written as pure, read-only, stateless functions that return data structures without producing side effects. The top-level `ResolveTurn()` orchestrator handles updating properties and injecting the definitive termination token (`MatchEndedEvent`) into the public stream, allowing AI modules to securely query hypothetical sandboxes without corrupting telemetry arrays.
- **Stateless Lounge-to-Game Manager**: Session lifetimes are managed via a `web.ServerStateManager`. The server starts in a "Lounge" state, dynamically allocates a match pointer (`engine.InitGame`) upon user action, and triggers a clean teardown (`ActiveMatch = nil`) upon match completion, returning to the Lounge.

## 6. Monolithic Architectural Separation & Presentation Boundaries

```text
+-------------------+      Uses Pointer (*)     +---------------------+
|  MatchController  | ------------------------> |     engine.Match    |
|   (UI Handler)    |                           | (Authoritative Core)|
+-------------------+                           +---------------------+
          |
          | Calls Contract
          v
+-------------------+
|   <<interface>>   |
|     MatchView     |
+-------------------+
          |
          +-----------------------+-----------------------+
          | (Phase 1)             | (Phase 2)             | (Phase 4 Future)
          v                       v                       v
+-------------------+   +-------------------+   +-------------------+
|   TerminalView    |   |      WebView      |   |   WebSocketView   |
|   (ASCII Text)    |   |  (JSON streaming) |   | (Live Event Pump) |
+-------------------+   +-------------------+   +-------------------+
```

- **Data-Isolated Presentation Boundary**: Decouples rendering from the core transaction state machine. To enforce turn secrecy and prevent sandbox memory leaks across boundaries, UI layouts receive an isolated pointer to `engine.GameState` (`WorkingState`) instead of the parent `engine.Match` orchestrator.
- **Abstract Rendering Contract (`MatchView`)**: Insulates input mechanics from layout layers via a stateless interface contract. Implementations satisfy compliance implicitly:
  - **TerminalView (Phase 1)**: Synchronously maps the 2D grid matrix into single-byte ASCII tokens (`█`, `B`, `U`) for raw terminal streaming.
  - **WebView (Phase 2)**: Directly serializes the active `GameState` struct into flat JSON arrays for HTTP response targets.
- **Flat Package Web Architecture**: Unlike the highly segregated `/engine` domain package, the `/server` package utilizes a cohesive flat file structure. This structure prevents circular dependencies during protocol handling and organizes network translation files by their specific communication protocol layer.
- **Pointer-Driven Memory Persistence (`*engine.Match`)**: Controller pipelines execute actions exclusively via `*engine.Match` references. This guarantees user interactions mutate master allocation frames natively, eliminating the memory duplication overhead of dynamic matrix slices.
- **Unified Service Entry Gate (`ApplyTurnCommand`)**: The engine exposes a single `ApplyTurnCommand` method. The MatchController (Phase 1) or HTTP handlers (Phase 2) construct `TurnCommand` structs with a `Type` discriminator (`TurnCmdMove`, `TurnCmdPlaceBomb`), allowing the engine to function as a stateless, decoupled command processor.

## 7. Network Sync, Transaction Pipeline & Idempotency Invariants

- **Localized Client-Side State Machine**: UI sub-phases—including tile discovery, reachability shading, and trajectory selection—execute strictly inside the client container to hit zero network overhead. Phase 1 leverages the local CLI controller; Phase 2 mirrors this via decoupled canvas logic utilizing static memory snapshots of the movement grid.
- **Atomic Command Transaction Execution**: Turn commitment passes actions through a single atomic `POST /api/match/actions` transaction. The backend parses the payload packet, pushes it into the master engine loop, applies verification rules, and drops a consolidated state bundle containing the evaluation snapshot and resolution log vectors.
- **Retained Playback Logs**: The sequence remains preserved throughout the turn resolution pass to drive localized client actions (raw terminal text traces in Phase 1; canvas animation frames and audio triggers in Phase 2). The array is wiped clean on the initialization boundary of a new turn sequence.
- **Idempotency and Self-Healing Synchronization**: Network drops and state desynchronizations drop down to zero-overhead overwriting loops. Because the engine treats every network interaction as an absolute state delivery event, the client handles incoming engine payloads as immutable truth, instantly wiping and rewriting local memory allocations without complex rolling diff checks.

## 8. Phase 2 HTTP API Design (REST)

### 8.1 Architecture Boundary

```
Client (Phaser.js / CLI)          Server (net/http)           Engine (pure Go)
────────────────────────────────────────────────────────────────────────────
Controls flow                     Mediates                    Pure calc
Renders/animates                  Validates requests          Rules, math
User input                        Serializes JSON             No I/O
                                  Manages rooms/matches       No time
                                  Calls engine methods
```

### 8.2 Turn Lifecycle (Client-Driven)

The server does **not** auto-call `StartTurn()` on match creation. Client explicitly starts each turn:

1. `POST /match-rooms/{id}/match` → `CreateMatch` (full GameCfg), returns 201
2. `GET /match-rooms/{id}/match/state` → clean Turn 1 state (no sudden death yet)
3. `POST /match-rooms/{id}/match/start-turn` → `StartTurn()` injects sudden death if `MaxTurns=0`
4. `GET /match/state` → state with bombs (animatable)
5. Planning: `POST /turn-commands`, `POST /reset` (sandbox)
6. `POST /resolve` → `ResolveTurn()`, returns events + next turn
7. Loop: `GET /state` → `POST /start-turn` → ...

Surrender: `POST /surrender` (either team, any time).

### 8.3 Key Decisions

- **State exposure**: `WorkingState` only (sandbox), never `TrueState`
- **Surrender**: Either team, validated as 1 or 2
- **Locking**: Global `mu` (Phase 2), per-room deferred to Phase 4
- **Stale cleanup**: 10min timeout, 30s interval, `LastActivity` updated on every request
- **CreateMatch**: Full GameCfg required (Phase 2); partial/join deferred to Phase 4
- **Player authorization with anonymous tokens**: Room IDs and UnitIDs are guessable. Any client with a room ID can impersonate any player. The solution is to adopt per-team cryptographically random tokens generated and stored in a new Match, returned once in `CreateMatchResponse`. Token will be validated in mutation endpoints. No restriction to read-only enpoints at the moment.

## 9. Phase 3 Frontend Toolchain & Decisions

- **Build Tool**: Vite provides native ESM and instant hot-module reloading (HMR), eliminating webpack-style rebuild delays during development.
- **Language**: TypeScript (strict mode) mirrors Go's compile-time safety and catches index/type errors before browser execution.
- **Game Framework**: Phaser 4.2.0 (breaking changes from v3: renderer, filter system, tint mechanism). Latest stable build compatible with procedural Graphics API.
- **Test Runner**: Vitest integrates with Vite, reuses Vite's config, maintains Jest-compatible APIs for rapid iteration.
- **Linting & Formatting**: ESLint + @typescript-eslint for static analysis, Prettier for deterministic code style.
- **Logical Resolution**: Fixed at `1280x720` to simplify UI layout math (16×9 aspect ratio).
- **Tile Size**: `48px` per cell provides visual clarity on standard desktop monitors while fitting 16x16 grids in viewport.
- **Mock Mode**: TitleScene includes an offline-play button for development (test game flow without backend connectivity).
- **Input Model**: Click-only interaction for Phase 3 (keyboard shortcuts deferred to Phase 5 polish pass).
- **Retro Art Strategy**: Phaser Graphics API generates all visuals procedurally (no sprite sheets required):
  - **Tiles**: Colored rectangles with borders.
  - **Units**: Geometric shapes (circles, triangles, pentagons, stars) tinted by team color.
  - **Bombs**: Filled circles with countdown text overlay.
  - **Soft Blocks**: Rounded rectangles.
  - **Explosions**: Tile color flash effect (simple visual feedback).


## File Structure (WIP)

```text
bomb-srpg
├── cli/                        <-- Phase 1: Interactive terminal CLI package
│   ├── match_controller.go     <-- Reads inputs and maps to engine actions
│   ├── views.go                <-- Defines the read-only MatchView interface
│   └── terminal_view.go        <-- Implements the ASCII map grid rendering logic
│
├── cmd/
│   ├── srpg-cli/               <-- Phase 1 Terminal entry point (v0.1.0-cli)
│   │   └── main.go
│   │
│   └── srpg-web/               <-- Phase 2+ HTTP entry point
│       └── main.go
│
├── server/                     <-- Phase 2: HTTP Web server package
│   ├── http_handlers.go        <-- REST HTTP interface boundary
│   ├── routes.go               <-- HTTP route registration
│   ├── server_manager.go       <-- Web server memory manager, state locks & housekeeper
│   └── ws_hub.go               <-- Phase 5: WebSocket connection event pump
│
├── docs/                       <-- Design, roadmap and other docs
├── engine/                     <-- Pure core logic
│   ├── codecs.go               <-- Bitmask encoders, decoders for UnitID and BombID
│   ├── commands.go             <-- TurnCommand types & constructors
│   ├── errors.go               <-- Error types deduced by engine
│   ├── events.go               <-- GameEvent types & constructors
│   ├── game.go                 <-- Game initializer
│   ├── match.go                <-- Match life cycle transactions
│   ├── models.go               <-- Pure blueprints
│   ├── pathfinding.go          <-- Stage navigation
│   ├── presets.go              <-- Static database
│   └── stage.go                <-- Centralized Stage verification and manipulation (IsInBound, ClearTile, UpdateTileOccupant, etc.)
│
├── Makefile                    <-- Build/Test Automation
└── web/public                  <-- Phase 3: Phaser.js Frontend UI
```

## Gameplay

### UX Lifecycle and Screen States

The game flow operates through a decoupled presentation layer managed entirely on the frontend via Phaser.js. The interface transitions through three distinct layout states:

1. **The Title Screen (Front Page)**
   - Displays game title logo (`Bomb Tactics`) and primary navigation routes.
   - User choices: `Match Mode` (Local vs. Human), `Online Mode` (Phase 4), `Story Mode` (Future Phase).

2. **The Match Lounge (Setup Screen)**
   - Triggered by selecting `Match Mode`.
   - Provides interface controllers to adjust configuration parameters before a game starts:
     - Map presets (Stages)
     - Max Turn limitations
     - Character Archetype choices for 2 Teams, etc.
   - Action Button: Clicking `[Start Game]` validates choices, bundles the parameters into a `GameCfg` JSON structure, and sends a `POST /api/match/create` network request to the Go backend room manager.

3. **The Active Gameplay Canvas**
   - Phaser.js captures the initialized `WorkingState` JSON reply from the server and instantly renders the 2D grid matrix world.
   - The user conducts their gameplay rounds step-by-step until an engine victory or surrender condition updates `WinnerTeamID != 0`.
   - Upon match resolution, the UI resets and routes the player cleanly back to the Match Lounge configuration state.

### In-Turn Action Economy Rules

To preserve strategic depth and prevent infinite execution exploits within a single sandbox turn planning cycle, each individual `Unit` is strictly bounded by a rigid action economy:

- **The Rule of 1-Move & 1-Bomb**: Within a single turn loop, an active character unit is permitted to execute a maximum of **one move action** and **one bomb placement action**.
- **Order Independent**: The execution sequence is completely flexible. A unit may choose to:
  - Move first, then drop a bomb.
  - Drop a bomb first, then move away.
  - Execute _only_ a move action or _only_ a bomb action.
  - Do nothing.
- **Sandbox Verification**: These status restrictions operate entirely within the engine's `WorkingState` scratchpad.
  - Executing a `/reset` system command completely restores a unit's action availability flags.
  - Transitioning via an authoritative `/commit` command completely flushes and refreshes these action limits back to zero inside `StartTurn()` for the upcoming round.
