import { bombID, softBlockID, unitID } from './ids';
import { SYSTEM_OWNER_ID } from '../api/types';
import type {
  Bomb,
  BombPlacedEvent,
  GameCfg,
  GameState,
  SoftBlock,
  TerrainType,
  Tile,
  Unit,
} from '../api/types';

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
    id: unitID(1),
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
  return {
    id: bombID(1),
    ownerId: unitID(1),
    position: { x: 0, y: 0 },
    range: 2,
    countdown: 3,
    ...overrides,
  };
}

export function makeSoftBlock(overrides: Partial<SoftBlock> = {}): SoftBlock {
  return { id: softBlockID(1), position: { x: 0, y: 0 }, ...overrides };
}

export function makeCfg(overrides: Partial<GameCfg> = {}): GameCfg {
  return {
    vsCpu: false,
    stagePreset: 'Plain',
    p1Slots: [],
    p2Slots: [],
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
    turnStartEvents: [],
    ...overrides,
  };
}

export function makeBombPlacedEvent(overrides: Partial<BombPlacedEvent> = {}): BombPlacedEvent {
  return {
    type: 'bombPlaced',
    unitId: SYSTEM_OWNER_ID,
    bombId: bombID(1),
    position: { x: 0, y: 0 },
    range: 2,
    countdown: 3,
    ...overrides,
  };
}
