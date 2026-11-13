export type TerrainType =
  'TerrainPlain' | 'TerrainBlock' | 'TerrainTower' | 'TerrainWater' | 'TerrainLava';

export type OccupantType =
  'OccupantNone' | 'OccupantUnit' | 'OccupantBomb' | 'OccupantSoftBlock' | 'OccupantItem';

export type SkillType = 'Jump' | 'Fly';

export type TurnCmdType = 'move' | 'placeBomb';

export type GameEvtType =
  | 'unitMoved'
  | 'unitDamaged'
  | 'unitDied'
  | 'bombPlaced'
  | 'bombCountdownUpdated'
  | 'bombExploded'
  | 'softBlockDestroyed'
  | 'matchEnded';

export type UnitRole = 'Normal' | 'King' | 'Boss';

export type CPUTurnPhase = 'TurnPhaseIdle' | 'TurnPhasePlanning' | 'TurnPhaseReady';

export interface Coordinate {
  x: number;
  y: number;
}

export interface Tile {
  type: TerrainType;
  occupantType: OccupantType;
  occupantId: number;
}

export interface SoftBlock {
  id: number;
  position: Coordinate;
  hiddenItem?: string;
}

export interface Archetype {
  name: string;
  speed: number;
  bombMaxRange: number;
  skills: SkillType[];
}

export interface StagePreset {
  name: string;
  description: string;
  width: number;
  height: number;
  maxTurns: number;
}

export interface Catalog {
  archetypes: Archetype[];
  stagePresets: StagePreset[];
}

export interface Unit {
  id: number;
  type: string;
  position: Coordinate;
  speed: number;
  bombMaxRange: number;
  bombPower: number;
  maxBombCount: number;
  bombUsed: number;
  team: number;
  hp: number;
  skills: SkillType[];
  hasMoved: boolean;
  hasUsedSkill: boolean;
  role: UnitRole;
}

export interface Bomb {
  id: number;
  ownerId: number;
  position: Coordinate;
  range: number;
  countdown: number;
}

export interface GameState {
  turn: number;
  inSuddenDeath: boolean;
  activeTeam: number;
  grid: Tile[][];
  units: Unit[];
  bombs: Bomb[];
  softBlocks: SoftBlock[];
  turnCommands: TurnCommand[];
  turnStartEvents: GameEvent[];
}

export interface TurnCommand {
  type: TurnCmdType;
  unitId: number;
  target: Coordinate;
}

export interface GameEvent {
  type: GameEvtType;
  unitId?: number;
  bombId?: number;
  softBlockId?: number;
  itemId?: number;
  position?: Coordinate;
  from?: Coordinate;
  to?: Coordinate;
  newHp?: number;
  range?: number;
  countdown?: number;
  affectedPositions?: Coordinate[];
  winnerTeamId?: number;
}

export interface TeamSlot {
  archetype: string;
  role: UnitRole;
}

export interface GameCfg {
  vsCpu: boolean;
  stagePreset: string;
  p1Slots: TeamSlot[];
  p2Slots: TeamSlot[];
  maxTurns: number;
  allowResetTurn: boolean;
}

export interface CreateMatchRoomResponse {
  id: string;
}

export interface CreateMatchRequest {
  gameCfg: GameCfg;
}

export interface CreateMatchResponse {
  success: boolean;
  playerTokens: [string, string];
}

export interface CpuStatusResponse {
  turnPhase: CPUTurnPhase;
  planGameEvents: GameEvent[];
  resolveTurnGameEvents: GameEvent[];
}

export interface SurrenderRequest {
  teamId: number;
}

export interface AllowedTilesRequest {
  unitId: number;
  turnCmdType: TurnCmdType;
}

export type AllowedTilesResponse = Coordinate[];

export interface ResolveTurnResponse {
  planGameEvents: GameEvent[];
  resolveTurnGameEvents: GameEvent[];
}
