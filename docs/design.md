# Technical Design Specification

## 1. Architectural Principles
- **Authoritative Server / Zero Trust**: The backend (`engine`) acts as the single source of truth. Incoming user command payloads are treated as unverified requests. The engine recalculates all pathfinding and rules independently before mutating memory.
- **State Sandboxing Pattern**: At the start of a turn, the engine clones the definitive `GameState` into a temporary `WorkingState` sandbox. All mid-turn actions (moving, placing bombs, picking up power-ups) alter this sandbox layer.
  - A `/reset` command discards the sandbox and re-clones the definitive checkpoint, wiping any accumulated power-up stat changes.
  - A `/commit` command promotes the `WorkingState` to the definitive state and triggers batch processing.
- **Batch Resolution**: All damage calculations, terrain status updates, bomb explosions, and chain reactions are deferred until a turn is committed. The resolution engine runs synchronously and returns a chronological `Animation Event Queue` JSON payload to the client.
- **Synchronous Execution Bound**: Given the low-density metrics (16x16 maximum matrix, <= 10 active characters), all grid lookups, BFS pathfinding, and explosion cascade evaluations are run synchronously to avoid multi-threaded race conditions.

## 2. Spatial Mapping & State Model Definitions
- **Grid Dimensions:** Dynamic N x M grid layout (supporting ranges from 7x7 up to a maximum bound of 16x16).
- **Storage Type:** Rows are allocated via dynamic Go slices (`[][]Cell`) to support custom asymmetrical maps without recompiling code.
- **Coordinate Mapping:** Layout is indexed as `Grid[Y][X]`. The top-left corner of the map is designated as `(0,0)`.
- **Dynamic Boundary Rule:** Every check evaluates dynamically against active bounds: `0 <= X < len(grid)` and `0 <= Y < len(grid)`.
- **Memory Normalization:** The Board Matrix tracks tile references via minimal integer identity tags (`OccupantID`). The GameState Engine keeps the master map directory of active entities. Moving a character updates two matrix cell structures, leaving core character stat records untouched.

## 3. Online Multiplayer Synchronisation (Phase 3a Choice)
- **Turn Secrecy Pattern:** Active turn planning is fully hidden from the opposing player to preserve the Turn Reset capability. Opponents see a passive waiting status during the planning phase.
- **Unified Event Broadcast:** Upon turn commitment, the backend generates an identical chronological Action Queue array and distributes it to both clients.
- **Frontend Queue Playback:** Clients process incoming batch payloads using a sequential async loop, ensuring both players watch animations unfold with perfect deterministic lockstep alignment.

# File Structure (Draft)

```text
bomb-srpg
├── cmd/
│   └── srpg-cli/          <-- Phase 1 use. Terminal CLI version
├── docs/                  <-- Design, roadmap and other docs
├── engine/                <-- Pure Core Logic
│   └── game.go            <-- Game initializer
│   └── match.go           <-- Match life cycle transactions
│   └── models.go          <-- Pure blueprints
│   └── pathfinder.go      <-- Stage navigation
│   └── presets.go         <-- Static database
└── web/                   <-- Frontend (Phase 2+)
```