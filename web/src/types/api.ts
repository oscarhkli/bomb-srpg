export type TerrainType =
  | 'TerrainPlain'
  | 'TerrainBlock'
  | 'TerrainTower'
  | 'TerrainWater'
  | 'TerrainLava';

export type OccupantType =
  | 'OccupantNone'
  | 'OccupantUnit'
  | 'OccupantBomb'
  | 'OccupantSoftBlock'
  | 'OccupantItem';

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

export interface Coordinate {
  x: number;
  y: number;
}

export interface Tile {
  type: TerrainType;
  occupantType: OccupantType;
  occupantId: number; // UnitID | BombID | SoftBlockID
}

export interface SoftBlock {
  id: number;
  position: Coordinate;
  hiddenItem?: string; // Reserved for future
}

export interface Archetype {
  name: string;
  speed: number;
  bombMaxRange: number;
  skills: SkillType[];
}

export interface Unit {
  id: number; // UnitID (uint8)
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
}

export interface Bomb {
  id: number; // BombID (uint32)
  ownerId: number; // UnitID (0 = sudden death)
  position: Coordinate;
  range: number;
  countdown: number;
}

/** Complete game state snapshot */
export interface GameState {
  turn: number;
  inSuddenDeath: boolean;
  activeTeam: number;
  grid: Tile[][];
  units: Unit[];
  bombs: Bomb[];
  softBlocks: SoftBlock[];
  turnCommands: TurnCommand[];
}

/** Player action during planning phase */
export interface TurnCommand {
  type: TurnCmdType;
  unitId: number;
  target: Coordinate;
}

// Optional fields vary by event type:
// unitMoved: from, to
// unitDamaged: newHp
// bombPlaced: bombId, position, range, countdown
// bombCountdownUpdated: bombId, countdown
// bombExploded: bombId, affectedPositions
// softBlockDestroyed: softBlockId, position
// matchEnded: winnerTeamId, isDraw
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

export interface GameCfg {
  stagePreset: string;
  p1Teams: string[]; // Archetype names (first = King)
  p2Teams: string[]; // Archetype names (first = King)
  maxTurns: number; // 0 = instant sudden death
  allowResetTurn: boolean;
  suddenDeath: boolean;
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

export interface SurrenderRequest {
  teamId: number;
}

export interface AllowedTilesRequest {
  unitId: number;
  turnCmdType: TurnCmdType;
}

export type AllowedTilesResponse = Coordinate[];

export interface StartTurnResponse {
  inSuddenDeath: boolean;
  gameEvents: GameEvent[];
}
