import { describe, it, expect } from 'vitest';
import { createMockGraphics, createMockText } from '../test/setup';
import type { GameState, Tile } from '../types/api';
import type { BombGraphics } from './resolveTurnPlayer';
import { cloneGrid, gridsEqual, occupantsMatch } from './stateSync';

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

describe('cloneGrid', () => {
  it('deep-copies tiles so mutating the clone leaves the original untouched', () => {
    const original: Tile[][] = [[tile({ occupantId: 7 })]];
    const copy = cloneGrid(original);

    copy[0]![0]!.occupantId = 99;

    expect(original[0]![0]!.occupantId).toBe(7);
    expect(copy[0]![0]!.occupantId).toBe(99);
  });
});

describe('gridsEqual', () => {
  it('is true for two structurally identical grids', () => {
    const a: Tile[][] = [[tile(), tile({ occupantType: 'OccupantUnit', occupantId: 7 })]];
    const b: Tile[][] = [[tile(), tile({ occupantType: 'OccupantUnit', occupantId: 7 })]];
    expect(gridsEqual(a, b)).toBe(true);
  });

  it('is false when a tile occupant differs', () => {
    const a: Tile[][] = [[tile({ occupantType: 'OccupantUnit', occupantId: 7 })]];
    const b: Tile[][] = [[tile({ occupantType: 'OccupantUnit', occupantId: 8 })]];
    expect(gridsEqual(a, b)).toBe(false);
  });

  it('is false when a tile terrain type differs', () => {
    expect(gridsEqual([[tile({ type: 'TerrainPlain' })]], [[tile({ type: 'TerrainLava' })]])).toBe(
      false
    );
  });

  it('is false when the grids have different dimensions', () => {
    expect(gridsEqual([[tile()]], [[tile(), tile()]])).toBe(false);
    expect(gridsEqual([[tile()]], [[tile()], [tile()]])).toBe(false);
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
