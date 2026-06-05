# Technical Design Specification

## 1. Architectural Principles

- **Authoritative Server / Zero Trust**: The backend (`engine`) acts as the single source of truth. Incoming user command payloads are treated as unverified requests. The engine recalculates all pathfinding and rules independently before mutating memory.
  * *Zero-Trust Parameters*: Property deductions—such as initial bomb countdowns based on terrain or player skills—are processed authoritatively on the server boundary. Client payloads cannot dictate property states.
- **State Sandboxing Pattern**: At the start of a turn, the engine clones the definitive `GameState` into a temporary `WorkingState` sandbox. All mid-turn actions (moving, placing bombs, picking up power-ups) alter this sandbox layer.
  * A `/reset` or `ResetTurn()` command discards the sandbox and re-clones the definitive checkpoint (`m.TrueState`), wiping any uncommitted player actions.
  * A `/commit` or `ResolveTurn()` command executes batch-resolution loops against the sandbox, advances the timeline, and permanently promotes the sandboxed state to the master history via a deep-copy clone.
* **Batch Resolution**: All damage calculations, terrain updates, bomb explosions, and chain reactions are deferred until a turn is committed. The resolution engine runs synchronously and returns a chronological `Animation Event Queue` string/binary array payload to the client.
- **Synchronous Execution Bound**: Given the low-density metrics (16x16 maximum matrix, <= 10 active characters), all grid lookups, BFS pathfinding, and explosion cascade evaluations are run synchronously to avoid multi-threaded race conditions.

## 2. Spatial Mapping & State Model Definitions

- **Stage Dimensions:** Dynamic N x M grid layout (supporting ranges from 7x7 up to a maximum bound of 16x16).
- **Storage Type:** Rows are allocated via dynamic Go slices (`[][]Tile`) to support custom asymmetrical maps without recompiling code.
- **Coordinate Mapping:** Layout is indexed as `Grid[Y][X]`. The top-left corner of the map is designated as `(0,0)`.
* **Dynamic Boundary Rule**: Every check evaluates dynamically against active bounds: `0 <= X < len(grid)` and `0 <= Y < len(grid)`.
- **Memory Normalization**: The Board Matrix tracks tile references via minimal structural fields (`OccupantType`, `OccupantID`). The GameState Engine maintains the master map directory of active entities. Moving an object updates only the cell metadata, leaving base entity metrics untouched.

## 3. Strongly Typed Identity Contracts & Bitmask Configurations

* **Compile-Time Contract Safety**: To guarantee type safety and eliminate argument-swapping bugs, core entity lookups are wrapped in custom, power-of-2 byte-aligned semantic primitives.
* **UnitID (uint8)**: Split internally into two equal 4-bit nibbles via `(TeamID << 4) | PlayerIndex`. This accommodates up to 15 unique teams and 15 players per team, mapping perfectly onto a single hexadecimal byte for clear log readability (e.g., `0x23` means Team 2, Player 3).
* **BombID (uint32)**: Packed to represent absolute timeline metadata using clear byte boundaries: `(UnitID << 24) | (CurrentTurn << 16) | BombCounterShift`.
* **TurnBombCounter Invariant**: A state-bound tracking sequence index that advances on the active sandbox frame. Because it is bound directly to the sandbox container, it resets on a turn-reset request, preventing ID collisions or frontend asset desynchronization.
* **SystemUnitID Contract**: The identifier `UnitID(0)` (Team 0, Player 0) is explicitly reserved as the authoritative environmental actor. It is utilized to log and process automated environmental events, such as sudden-death hazards, without creating phantom player models.

## 4. Pathfinding, Trajectory Vectors & Collision Engine

- **Transient Snapshot Pattern**: Instead of heavy OOP decorator patterns or permanent state mutations, movement capabilities are passed into pathfinding as a short-lived, discardable `MovementRule` context instruction slip generated on the fly using active character statistics.
- **High-Performance Bitmask Matrix**: Entity pass-through permissions are packed into a highly optimized, 1-byte bitmask field (`PassFlags` uint8). This completely eliminates nested slice iteration loops inside the pathfinder, collapsing obstacle evaluation down to a bitwise operation.
* **Separation of Concerns (Reachability vs. Legality)**:
  * The pathfinder function (`FindReachableTiles`) has exactly one job: mapping spatial reachability and step distance tracking (`map[Coordinate]int`), treating the origin tile as step 0.
  * Target landing restrictions (e.g., landing on an item tile or targeting allies) are business rules evaluated independently by respective **Action Handlers** *after* pathfinding returns.
* **Unified Impact Absorption Crucible**: Straight-line calculation rays (character walking, bomb explosions) and corner-wrapping paths (sanity check). An explicit impact flag (`StopOnFirstNonUnitOccupant`) forces the ray to register a valid hit on a solid obstacle (SoftBlock, Bomb, Item) but instantly terminates the vector to shield cells behind it.
* **Snapshot-Based Detonation**: All cascading bomb explosions within an intra-turn phase resolve at the exact same physical millisecond. To prevent ray-truncation anomalies, the engine queries a read-only 2D snapshot copy of the board captured at the start of the resolution pass (`cloneGridSnapshot()`). Destructible soft blocks flag their destruction inside delayed registries but remain solid, ray-blocking obstacles until the end of the pass to ensure perfect unit shielding.
* **Intra-Turn Damage Capping**: Units caught in overlapping blast patterns or multiple cascading explosions lose a flat maximum of exactly 1 HP, matching the low-density 1-HP character pacing rules. Damage calculations utilize a local boolean presence set to record injuries rather than an accumulator, preventing health overflow. All entity modifications, soft wall dissolutions, and tile clear passes execute as a simultaneous batch flush at the very end of the loop.

## 5. Online Multiplayer Synchronisation & State Machine Loops

- **Turn Secrecy Pattern:** Active turn planning is fully hidden from the opposing player to preserve the Turn Reset capability. Opponents see a passive waiting status during the planning phase.
- **Unified Event Broadcast:** Upon turn commitment, the backend generates an identical chronological Action Queue array and distributes it to both clients.
- **Frontend Queue Playback:** Clients process incoming batch payloads using a sequential async loop, ensuring both players watch animations unfold with perfect deterministic lockstep alignment.
* **Turn Startup Sudden Death Checks**: State machine transition and boundary rules—such as checking if `TrueState.Turn >= Config.MaxTurn`—are evaluated at the very beginning of a new turn (`StartNewTurn()`). This ensures map alterations and automated sudden-death bomb injections are fully populated and rendered before a player can input commands. Setting `MaxTurn = 0` forces instant sudden death on Turn 1.
* **Decoupled Analytical Queries**: Victory evaluations (`EvaluateVictoryConditions()`) are written as pure, read-only, stateless functions that return data structures without producing side effects. The top-level `ResolveTurn()` orchestrator handles updating properties and injecting the definitive termination token (`MatchEndedEvent`) into the public stream, allowing AI modules to securely query hypothetical sandboxes without corrupting telemetry arrays.

# File Structure (WIP)

```text
bomb-srpg
├── cmd/
│   └── cli/                    <-- Phase 1: Interactive Terminal CLI Driver
│       ├── main.go             <-- Initialises engine and drives the 1-time render
│       ├── match_controller.go <-- Reads inputs and maps to engine actions
│       ├── views.go            <-- Defines the read-only MatchView interface
│       └── terminal_view.go    <-- Implements the ASCII map grid rendering logic
├── srpg-cli.go                 <-- Phase 1 entry point
├── srpg-cli.go                 <-- Phase 2 entry point
├── docs/                       <-- Design, roadmap and other docs
├── engine/                     <-- Pure Core Logic
│   └── codecs.go               <-- Bitmask encoders, decoders for UnitID and BombID
│   └── game.go                 <-- Game initializer
│   └── match.go                <-- Match life cycle transactions
│   └── models.go               <-- Pure blueprints
│   └── pathfinding.go          <-- Stage navigation
│   └── presets.go              <-- Static database
|   └── stage.go                <-- Centralized Stage verification and manipulation (IsInBound, ClearTile, UpdateTileOccupant, etc.)
└── web/                        <-- Phase 2+: Frontend
```
