# Technical Design Specification

## 1. Architectural Principles
- **Authoritative Server / Zero Trust**: The backend (`engine`) acts as the single source of truth. Incoming user command payloads are treated as unverified requests. The engine recalculates all pathfinding and rules independently before mutating memory.
- **State Sandboxing Pattern**: At the start of a turn, the engine clones the definitive `GameState` into a temporary `WorkingState` sandbox. All mid-turn actions (moving, placing bombs, picking up power-ups) alter this sandbox layer.
  - A `/reset` command discards the sandbox and re-clones the definitive checkpoint, wiping any accumulated power-up stat changes.
  - A `/commit` command promotes the `WorkingState` to the definitive state and triggers batch processing.
- **Batch Resolution**: All damage calculations, terrain status updates, bomb explosions, and chain reactions are deferred until a turn is committed. The resolution engine runs synchronously and returns a chronological `Animation Event Queue` JSON payload to the client.
- **Synchronous Execution Bound**: Given the low-density metrics (16x16 maximum matrix, <= 10 active characters), all grid lookups, BFS pathfinding, and explosion cascade evaluations are run synchronously to avoid multi-threaded race conditions.

## 2. Spatial Mapping & State Model Definitions
- **Stage Dimensions:** Dynamic N x M grid layout (supporting ranges from 7x7 up to a maximum bound of 16x16).
- **Storage Type:** Rows are allocated via dynamic Go slices (`[][]Tile`) to support custom asymmetrical maps without recompiling code.
- **Coordinate Mapping:** Layout is indexed as `Grid[Y][X]`. The top-left corner of the map is designated as `(0,0)`.
- **Dynamic Boundary Rule:** Every check evaluates dynamically against active bounds: `0 <= X < len(grid)` and `0 <= Y < len(grid)`.
- **Memory Normalization**: The Board Matrix tracks tile references via minimal structural fields (`OccupantType`, `OccupantID`). The GameState Engine maintains the master map directory of active entities. Moving an object updates only the cell metadata, leaving base entity metrics untouched.

## 3. Pathfinding, Trajectory Vectors & Collision Engine
- **Transient Snapshot Pattern**: Instead of heavy OOP decorator patterns or permanent state mutations, movement capabilities are passed into pathfinding as a short-lived, discardable `MovementRule` context instruction slip generated on the fly using active character statistics.
- **High-Performance Bitmask Matrix**: Entity pass-through permissions are packed into a highly optimized, 1-byte bitmask field (`PassFlags` uint8). This completely eliminates nested slice iteration loops inside the pathfinder, collapsing obstacle evaluation down to a bitwise operation.
- **Separation of Concerns (Reachability vs. Legality)**: 
  - The pathfinder function (`FindReachableTiles`) has exactly one job: mapping **spatial reachability and step distance tracking** (`map[Coordinate]int`). It treats the origin tile as step 0.
  - Target landing restrictions (e.g., "Can a bomb land on an item tile?", "Can a unit use a skill only on their allies?") are business rules evaluated independently by the respective **Action Handlers** *after* pathfinding returns.
- **Unified Impact Absorption Crucible**: Straight-line calculation rays (bomb explosions) and corner-wrapping paths (character walking) use a single, flattened `if-continue` guard loop. 
  - An explicit impact flag (`StopOnFirstNonUnitOccupant`) forces the ray to enter a cell containing a solid obstacle (SoftBlock, Bomb, Item) to register a valid hit, but instantly terminates the directional queue vector to shield cells behind it.

## 4. Online Multiplayer Synchronisation
- **Turn Secrecy Pattern:** Active turn planning is fully hidden from the opposing player to preserve the Turn Reset capability. Opponents see a passive waiting status during the planning phase.
- **Unified Event Broadcast:** Upon turn commitment, the backend generates an identical chronological Action Queue array and distributes it to both clients.
- **Frontend Queue Playback:** Clients process incoming batch payloads using a sequential async loop, ensuring both players watch animations unfold with perfect deterministic lockstep alignment.

# File Structure (WIP)

```text
bomb-srpg
├── cmd/
│   └── srpg-cli/          <-- Phase 1: Interactive Terminal CLI Driver
├── docs/                  <-- Design, roadmap and other docs
├── engine/                <-- Pure Core Logic
│   └── game.go            <-- Game initializer
│   └── match.go           <-- Match life cycle transactions
│   └── models.go          <-- Pure blueprints
│   └── pathfinding.go     <-- Stage navigation
│   └── presets.go         <-- Static database
|   └── stage.go           <-- Centralized Stage verification and manipulation (IsInBound, ClearTile, UpdateTileOccupant, etc.)
└── web/                   <-- Phase 2+: Frontend
```