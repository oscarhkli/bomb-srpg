// Shared domain-object builders for tests. Pure data — no Phaser/mockScene dependency, so any
// test file (including non-rendering ones like api.test.ts) can import from here.
import type {
  Bomb,
  GameCfg,
  GameEvent,
  GameState,
  SoftBlock,
  TerrainType,
  Tile,
  Unit,
} from '../types/api';

export function tileOf(type: TerrainType): Tile {
  return { type, occupantType: 'OccupantNone', occupantId: 0 };
}

export function plainTile(): Tile {
  return tileOf('TerrainPlain');
}

export function plainGrid(rows: number, cols: number): Tile[][] {
  return Array.from({ length: rows }, () => Array.from({ length: cols }, plainTile));
}

export function makeUnit(overrides: Partial<Unit> = {}): Unit {
  return {
    id: 1,
    type: 'Fighter',
    position: { x: 0, y: 0 },
    speed: 2,
    bombMaxRange: 2,
    bombPower: 1,
    maxBombCount: 3,
    bombUsed: 0,
    team: 1,
    hp: 1,
    skills: [],
    hasMoved: false,
    hasUsedSkill: false,
    ...overrides,
  };
}

export function makeBomb(overrides: Partial<Bomb> = {}): Bomb {
  return { id: 1, ownerId: 1, position: { x: 0, y: 0 }, range: 2, countdown: 3, ...overrides };
}

export function makeSoftBlock(overrides: Partial<SoftBlock> = {}): SoftBlock {
  return { id: 1, position: { x: 0, y: 0 }, ...overrides };
}

export function makeCfg(overrides: Partial<GameCfg> = {}): GameCfg {
  return {
    stagePreset: 'default',
    p1Teams: [],
    p2Teams: [],
    maxTurns: 30,
    allowResetTurn: true,
    ...overrides,
  };
}

export function makeState(overrides: Partial<GameState> = {}): GameState {
  return {
    turn: 1,
    inSuddenDeath: false,
    activeTeam: 1,
    grid: [[plainTile()]],
    units: [],
    bombs: [],
    softBlocks: [],
    turnCommands: [],
    ...overrides,
  };
}

export function makeBombPlacedEvent(overrides: Partial<GameEvent> = {}): GameEvent {
  return {
    type: 'bombPlaced',
    unitId: 0,
    bombId: 1,
    position: { x: 0, y: 0 },
    range: 2,
    countdown: 3,
    ...overrides,
  };
}
