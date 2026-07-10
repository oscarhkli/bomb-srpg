import { describe, it, expect } from 'vitest';
import { createMockGraphics, createMockText } from '../test/setup';
import type { GameEvent, GameState, Tile, TurnCommand } from '../types/api';
import type { BombGraphics } from './resolveTurnPlayer';
import { extractAppliedTarget, turnCommandTargetMatches, occupantsMatch } from './stateSync';

function tile(overrides: Partial<Tile> = {}): Tile {
  return { type: 'TerrainPlain', occupantType: 'OccupantNone', occupantId: 0, ...overrides };
}

function state(overrides: Partial<GameState> = {}): GameState {
  return {
    turn: 1,
    activeTeam: 1,
    grid: [[tile()]],
    units: [],
    bombs: [],
    softBlocks: [],
    turnCommands: [],
    ...overrides,
  };
}

function bombGraphics(): BombGraphics {
  return {
    circle: createMockGraphics() as never,
    countdownText: createMockText() as never,
  };
}

describe('extractAppliedTarget', () => {
  it('returns the server-reported `to` for a move command, not the requested target', () => {
    const cmd: TurnCommand = { type: 'move', unitId: 7, target: { x: 1, y: 0 } };
    const events: GameEvent[] = [
      { type: 'unitMoved', unitId: 7, from: { x: 0, y: 0 }, to: { x: 2, y: 0 } },
    ];
    expect(extractAppliedTarget(cmd, events)).toEqual({
      type: 'move',
      unitId: 7,
      to: { x: 2, y: 0 },
    });
  });

  it('returns undefined for a move command when no unitMoved event is present', () => {
    const cmd: TurnCommand = { type: 'move', unitId: 7, target: { x: 1, y: 0 } };
    expect(extractAppliedTarget(cmd, [])).toBeUndefined();
  });

  it('returns the server-assigned bombId and position for a placeBomb command', () => {
    const cmd: TurnCommand = { type: 'placeBomb', unitId: 7, target: { x: 1, y: 0 } };
    const events: GameEvent[] = [
      { type: 'bombPlaced', unitId: 7, bombId: 42, position: { x: 1, y: 0 }, countdown: 3 },
    ];
    expect(extractAppliedTarget(cmd, events)).toEqual({
      type: 'placeBomb',
      bombId: 42,
      position: { x: 1, y: 0 },
    });
  });

  it('returns undefined for a placeBomb command when no bombPlaced event is present', () => {
    const cmd: TurnCommand = { type: 'placeBomb', unitId: 7, target: { x: 1, y: 0 } };
    expect(extractAppliedTarget(cmd, [])).toBeUndefined();
  });
});

describe('turnCommandTargetMatches', () => {
  it('is true for a move result when the unit exists at the reported `to` position', () => {
    const result = { type: 'move' as const, unitId: 7, to: { x: 1, y: 0 } };
    const fresh = state({ units: [{ id: 7, position: { x: 1, y: 0 } } as never] });
    expect(turnCommandTargetMatches(fresh, result)).toBe(true);
  });

  it('is false for a move result when the unit is missing from fresh state', () => {
    const result = { type: 'move' as const, unitId: 7, to: { x: 1, y: 0 } };
    expect(turnCommandTargetMatches(state(), result)).toBe(false);
  });

  it('is false for a move result when the unit exists but at a different position', () => {
    const result = { type: 'move' as const, unitId: 7, to: { x: 1, y: 0 } };
    const fresh = state({ units: [{ id: 7, position: { x: 0, y: 0 } } as never] });
    expect(turnCommandTargetMatches(fresh, result)).toBe(false);
  });

  it('is true for a placeBomb result when a bomb with matching bombId exists at the reported position', () => {
    const result = { type: 'placeBomb' as const, bombId: 42, position: { x: 1, y: 0 } };
    const fresh = state({ bombs: [{ id: 42, ownerId: 7, position: { x: 1, y: 0 } } as never] });
    expect(turnCommandTargetMatches(fresh, result)).toBe(true);
  });

  it('is false for a placeBomb result when no bomb with matching bombId exists at the reported position', () => {
    const result = { type: 'placeBomb' as const, bombId: 42, position: { x: 1, y: 0 } };
    expect(turnCommandTargetMatches(state(), result)).toBe(false);
  });
});

describe('occupantsMatch', () => {
  it('is true when live units, bombs, and softBlocks all have a matching graphics entry', () => {
    const fresh = state({
      units: [{ id: 1, hp: 1 } as never, { id: 2, hp: 1 } as never],
      bombs: [{ id: 10 } as never],
      softBlocks: [{ id: 20 } as never],
    });

    const match = occupantsMatch(
      fresh,
      new Map([
        [1, createMockGraphics() as never],
        [2, createMockGraphics() as never],
      ]),
      new Map([[10, bombGraphics()]]),
      new Map([[20, createMockGraphics() as never]])
    );

    expect(match).toBe(true);
  });

  it('ignores dead units (hp 0) when comparing against the unit graphics map', () => {
    const fresh = state({
      units: [{ id: 1, hp: 1 } as never, { id: 2, hp: 0 } as never],
    });

    // Only the live unit (id 1) has a graphics entry — the dead unit must not count.
    expect(
      occupantsMatch(fresh, new Map([[1, createMockGraphics() as never]]), new Map(), new Map())
    ).toBe(true);
  });

  it('is false when a live unit has no graphics entry', () => {
    const fresh = state({ units: [{ id: 1, hp: 1 } as never] });
    expect(occupantsMatch(fresh, new Map(), new Map(), new Map())).toBe(false);
  });

  it('is false when a bomb has no graphics entry', () => {
    const fresh = state({ bombs: [{ id: 10 } as never] });
    expect(occupantsMatch(fresh, new Map(), new Map(), new Map())).toBe(false);
  });

  it('is false when a softBlock has no graphics entry', () => {
    const fresh = state({ softBlocks: [{ id: 20 } as never] });
    expect(occupantsMatch(fresh, new Map(), new Map(), new Map())).toBe(false);
  });
});
