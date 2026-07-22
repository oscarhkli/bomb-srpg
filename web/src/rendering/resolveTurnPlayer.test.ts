import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene, createMockGraphics, createMockText, createMockContainer } from '../test/setup';
import { delayedCallAt, makeBombGraphics } from '../test/sceneHelpers';
import { makeState, makeUnit, makeSoftBlock, plainGrid } from '../test/fixtures';
import { occupantsMatch } from '../test/occupantsMatch';
import type { GameEvent, GameState } from '../types/api';
import { BLAST_SPEED_MS_PER_TILE, BLAST_DURATION_MS, FIRE_DURATION_MS } from './constants';
import { TILE_SIZE } from '../constants';
import { playResolveTurnEvents, type BombGraphics } from './resolveTurnPlayer';

beforeEach(() => {
  vi.clearAllMocks();
});

function baseState(overrides: Partial<GameState> = {}): GameState {
  return makeState({
    grid: [[{ type: 'TerrainPlain', occupantType: 'OccupantBomb', occupantId: 1 }]],
    bombs: [{ id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 1, countdown: 2 }],
    ...overrides,
  });
}

function grid5x5(): GameState['grid'] {
  return plainGrid(5, 5);
}

describe('playResolveTurnEvents — bombCountdownUpdated', () => {
  it('updates the bomb countdown text immediately (offset 0) when countdown > 0', () => {
    const countdownText = createMockText();
    const bombGraphicsById = new Map<number, BombGraphics>([
      [
        1,
        {
          container: createMockContainer() as never,
          countdownText: countdownText as never,
        },
      ],
    ]);

    const result = playResolveTurnEvents(
      [{ type: 'bombCountdownUpdated', bombId: 1, countdown: 3 }],
      {
        scene: mockScene as never,
        gameStateSnapshot: baseState(),
        unitGraphicsById: new Map(),
        bombGraphicsById,
        softBlockGraphicsById: new Map(),
        onError: vi.fn(),
      }
    );

    expect(result.ok).toBe(true);
    delayedCallAt(0)();
    expect(countdownText.setText).toHaveBeenCalledWith('3');
  });

  it('renders a red "!" when countdown reaches 0', () => {
    const countdownText = createMockText();
    const bombGraphicsById = new Map<number, BombGraphics>([
      [
        1,
        {
          container: createMockContainer() as never,
          countdownText: countdownText as never,
        },
      ],
    ]);

    playResolveTurnEvents([{ type: 'bombCountdownUpdated', bombId: 1, countdown: 0 }], {
      scene: mockScene as never,
      gameStateSnapshot: baseState(),
      unitGraphicsById: new Map(),
      bombGraphicsById,
      softBlockGraphicsById: new Map(),
      onError: vi.fn(),
    });

    delayedCallAt(0)();
    expect(countdownText.setText).toHaveBeenCalledWith('!');
    expect(countdownText.setColor).toHaveBeenCalledWith('#ff0000');
  });

  it('does not throw when the bomb graphics map has no entry for a validated bombId', () => {
    // bombGraphicsById deliberately left empty — validate() only checks the snapshot,
    // not that the graphics map is in sync with it.
    playResolveTurnEvents([{ type: 'bombCountdownUpdated', bombId: 1, countdown: 3 }], {
      scene: mockScene as never,
      gameStateSnapshot: baseState(),
      unitGraphicsById: new Map(),
      bombGraphicsById: new Map(),
      softBlockGraphicsById: new Map(),
      onError: vi.fn(),
    });

    expect(() => delayedCallAt(0)()).not.toThrow();
  });
});

describe('playResolveTurnEvents — validation', () => {
  it('flags a missing required field and schedules nothing', () => {
    const onError = vi.fn();

    const result = playResolveTurnEvents([{ type: 'bombCountdownUpdated', bombId: 1 }], {
      scene: mockScene as never,
      gameStateSnapshot: baseState(),
      unitGraphicsById: new Map(),
      bombGraphicsById: new Map(),
      softBlockGraphicsById: new Map(),
      onError,
    });

    expect(result.ok).toBe(false);
    expect(onError).toHaveBeenCalledOnce();
    expect(mockScene.time.delayedCall).not.toHaveBeenCalled();
  });

  it('flags a bombId that does not exist in the snapshot and schedules nothing', () => {
    const onError = vi.fn();

    const result = playResolveTurnEvents(
      [{ type: 'bombCountdownUpdated', bombId: 999, countdown: 1 }],
      {
        scene: mockScene as never,
        gameStateSnapshot: baseState(),
        unitGraphicsById: new Map(),
        bombGraphicsById: new Map(),
        softBlockGraphicsById: new Map(),
        onError,
      }
    );

    expect(result.ok).toBe(false);
    expect(onError).toHaveBeenCalledOnce();
    expect(mockScene.time.delayedCall).not.toHaveBeenCalled();
  });

  it('flags a negative countdown as an invalid value and schedules nothing', () => {
    const onError = vi.fn();

    const result = playResolveTurnEvents(
      [{ type: 'bombCountdownUpdated', bombId: 1, countdown: -1 }],
      {
        scene: mockScene as never,
        gameStateSnapshot: baseState(),
        unitGraphicsById: new Map(),
        bombGraphicsById: new Map(),
        softBlockGraphicsById: new Map(),
        onError,
      }
    );

    expect(result.ok).toBe(false);
    expect(onError).toHaveBeenCalledOnce();
    expect(mockScene.time.delayedCall).not.toHaveBeenCalled();
  });

  it('flags a negative newHp as an invalid value and schedules nothing', () => {
    const onError = vi.fn();
    const state = baseState({
      units: [makeUnit({ id: 0x21, type: 'Bandit', speed: 3, maxBombCount: 1, team: 2 })],
    });

    const result = playResolveTurnEvents([{ type: 'unitDamaged', unitId: 0x21, newHp: -1 }], {
      scene: mockScene as never,
      gameStateSnapshot: state,
      unitGraphicsById: new Map(),
      bombGraphicsById: new Map(),
      softBlockGraphicsById: new Map(),
      onError,
    });

    expect(result.ok).toBe(false);
    expect(onError).toHaveBeenCalledOnce();
    expect(mockScene.time.delayedCall).not.toHaveBeenCalled();
  });

  it('flags a bombExploded affected position that is out-of-bounds', () => {
    const onError = vi.fn();
    const state = baseState({
      grid: grid5x5(),
      bombs: [{ id: 1, ownerId: 0x11, position: { x: 2, y: 2 }, range: 2, countdown: 0 }],
    });

    const result = playResolveTurnEvents(
      [{ type: 'bombExploded', bombId: 1, affectedPositions: [{ x: 99, y: 99 }] }],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map(),
        bombGraphicsById: new Map(),
        softBlockGraphicsById: new Map(),
        onError,
      }
    );

    expect(result.ok).toBe(false);
    expect(onError).toHaveBeenCalledOnce();
    expect(mockScene.time.delayedCall).not.toHaveBeenCalled();
  });

  it('accepts an in-bounds bombExploded affected position that is not cardinally aligned with the bomb, rendering no beam for it', () => {
    const onError = vi.fn();
    const bombGraphicsById = new Map<number, BombGraphics>([[1, makeBombGraphics()]]);
    const state = baseState({
      grid: grid5x5(),
      bombs: [{ id: 1, ownerId: 0x11, position: { x: 2, y: 2 }, range: 2, countdown: 0 }],
    });

    const result = playResolveTurnEvents(
      // (3,3) is diagonal to the bomb at (2,2) — not on the same row or column, but in-bounds.
      // A future engine change could plausibly emit this; it must not be rejected client-side.
      [{ type: 'bombExploded', bombId: 1, affectedPositions: [{ x: 3, y: 3 }] }],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map(),
        bombGraphicsById,
        softBlockGraphicsById: new Map(),
        onError,
      }
    );

    expect(result.ok).toBe(true);
    expect(onError).not.toHaveBeenCalled();
    delayedCallAt(0)();
    // The bomb graphics are still cleaned up, but no beam is drawn since the only
    // affected position has no cardinal direction from the bomb.
    expect(mockScene.tweens.add).not.toHaveBeenCalled();
  });
});

describe('playResolveTurnEvents — bombExploded (non-chain)', () => {
  it('destroys the bomb graphics immediately and grows a beam toward the affected tiles', () => {
    const container = createMockContainer();
    const countdownText = createMockText();
    const bombGraphicsById = new Map<number, BombGraphics>([
      [
        1,
        {
          container: container as never,
          countdownText: countdownText as never,
        },
      ],
    ]);
    const state = baseState({
      grid: grid5x5(),
      bombs: [{ id: 1, ownerId: 0x11, position: { x: 2, y: 2 }, range: 2, countdown: 0 }],
    });

    const result = playResolveTurnEvents(
      [
        {
          type: 'bombExploded',
          bombId: 1,
          affectedPositions: [
            { x: 3, y: 2 },
            { x: 4, y: 2 },
          ],
        },
      ],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map(),
        bombGraphicsById,
        softBlockGraphicsById: new Map(),
        onError: vi.fn(),
      }
    );

    expect(result.ok).toBe(true);

    delayedCallAt(0)();
    expect(container.destroy).toHaveBeenCalled();
    expect(bombGraphicsById.has(1)).toBe(false);

    expect(mockScene.tweens.add).toHaveBeenCalledOnce();
    const tweenCfg = mockScene.tweens.add.mock.calls[0]![0] as { len: number; duration: number };
    expect(tweenCfg.len).toBe(2 * TILE_SIZE); // 2 tiles east
    expect(tweenCfg.duration).toBe(2 * BLAST_SPEED_MS_PER_TILE);
  });

  it("accepts the bomb's own origin tile in affectedPositions (the backend always includes it) and still renders", () => {
    const onError = vi.fn();
    const bombGraphicsById = new Map<number, BombGraphics>([[1, makeBombGraphics()]]);
    const state = baseState({
      grid: grid5x5(),
      bombs: [{ id: 1, ownerId: 0x11, position: { x: 2, y: 2 }, range: 2, countdown: 0 }],
    });

    const result = playResolveTurnEvents(
      [
        {
          type: 'bombExploded',
          bombId: 1,
          // The engine's raycast seeds the reachable set with the bomb's own tile,
          // so (2,2) is always present alongside the outward ray tiles.
          affectedPositions: [
            { x: 2, y: 2 },
            { x: 3, y: 2 },
            { x: 4, y: 2 },
          ],
        },
      ],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map(),
        bombGraphicsById,
        softBlockGraphicsById: new Map(),
        onError,
      }
    );

    expect(result.ok).toBe(true);
    expect(onError).not.toHaveBeenCalled();
    // The origin tile no longer aborts rendering — the outward beam is still scheduled
    // once the explosion's delayedCall(0) fires.
    delayedCallAt(0)();
    expect(mockScene.tweens.add).toHaveBeenCalledOnce();
  });
});

describe('playResolveTurnEvents — chain reactions', () => {
  it('delays a chain-reacted bomb by reachTime from the causing bomb whose blast reached it', () => {
    const bombGraphicsById = new Map<number, BombGraphics>([
      [1, makeBombGraphics()],
      [2, makeBombGraphics()],
    ]);
    const state = baseState({
      grid: grid5x5(),
      bombs: [
        { id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 2, countdown: 0 },
        { id: 2, ownerId: 0x11, position: { x: 2, y: 0 }, range: 1, countdown: 2 },
      ],
    });

    playResolveTurnEvents(
      [
        {
          type: 'bombExploded',
          bombId: 1,
          affectedPositions: [
            { x: 1, y: 0 },
            { x: 2, y: 0 },
          ],
        },
        { type: 'bombExploded', bombId: 2, affectedPositions: [{ x: 3, y: 0 }] },
      ],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map(),
        bombGraphicsById,
        softBlockGraphicsById: new Map(),
        onError: vi.fn(),
      }
    );

    // bomb 2 sits at (2,0), which is in bomb 1's affectedPositions => chain reaction,
    // delayed by reachTime((0,0), (2,0)) = 2 tiles * 60ms = 120ms from bomb 1's blast-start (0).
    const chainDelays = mockScene.time.delayedCall.mock.calls
      .map(c => c[0] as number)
      .filter(d => d === 120);
    expect(chainDelays.length).toBeGreaterThan(0);
  });

  it('picks the smallest resulting delay when a position is covered by more than one earlier blast, even if that blast is not the earliest in the array', () => {
    const bombGraphicsById = new Map<number, BombGraphics>([
      [1, makeBombGraphics()],
      [2, makeBombGraphics()],
      [3, makeBombGraphics()],
    ]);
    const state = baseState({
      grid: grid5x5(),
      bombs: [
        { id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 3, countdown: 0 },
        { id: 2, ownerId: 0x11, position: { x: 2, y: 0 }, range: 1, countdown: 0 },
        { id: 3, ownerId: 0x11, position: { x: 3, y: 0 }, range: 1, countdown: 2 },
      ],
    });

    playResolveTurnEvents(
      [
        // Bomb 1 (earliest in array, but farther): reaches (3,0) via 3 tiles => 180ms.
        {
          type: 'bombExploded',
          bombId: 1,
          affectedPositions: [
            { x: 1, y: 0 },
            { x: 3, y: 0 },
          ],
        },
        // Bomb 2 (later in array, but closer): reaches (3,0) via 1 tile => 60ms.
        { type: 'bombExploded', bombId: 2, affectedPositions: [{ x: 3, y: 0 }] },
        // Bomb 3 sits at (3,0) — a chain reaction caused by both 1 and 2.
        { type: 'bombExploded', bombId: 3, affectedPositions: [{ x: 4, y: 0 }] },
      ],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map(),
        bombGraphicsById,
        softBlockGraphicsById: new Map(),
        onError: vi.fn(),
      }
    );

    // Smallest-delay tie-break must pick bomb 2's 60ms, not bomb 1's (earlier-in-array) 180ms.
    const delays = mockScene.time.delayedCall.mock.calls.map(c => c[0] as number);
    expect(delays).toContain(60);
    expect(delays).not.toContain(180);
  });
});

describe('playResolveTurnEvents — occupant events', () => {
  it('delays unitDamaged by reachTime from the causing bomb, using the snapshot position not the event position', () => {
    const unitGraphics = createMockGraphics();
    const state = baseState({
      grid: grid5x5(),
      bombs: [{ id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 2, countdown: 0 }],
      units: [
        makeUnit({
          id: 0x21,
          type: 'Bandit',
          position: { x: 2, y: 0 },
          speed: 3,
          maxBombCount: 1,
          team: 2,
        }),
      ],
    });

    playResolveTurnEvents(
      [
        {
          type: 'bombExploded',
          bombId: 1,
          affectedPositions: [
            { x: 1, y: 0 },
            { x: 2, y: 0 },
          ],
        },
        // Deliberately-wrong `position` on the wire — snapshot position (2,0) must win.
        { type: 'unitDamaged', unitId: 0x21, newHp: 0, position: { x: 9, y: 9 } },
      ],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map([[0x21, unitGraphics as never]]),
        bombGraphicsById: new Map([[1, makeBombGraphics()]]),
        softBlockGraphicsById: new Map(),
        onError: vi.fn(),
      }
    );

    // reachTime((0,0), (2,0)) = 2 * 60 = 120ms.
    const delays = mockScene.time.delayedCall.mock.calls.map(c => c[0] as number);
    expect(delays).toContain(120);
  });

  it('keeps a dying unit burning until unitDied fires, 3s after which unit + fire are removed', () => {
    const unitGraphics = createMockGraphics();
    const state = baseState({
      grid: grid5x5(),
      units: [
        makeUnit({
          id: 0x21,
          type: 'Bandit',
          position: { x: 1, y: 1 },
          speed: 3,
          maxBombCount: 1,
          team: 2,
          hp: 0,
        }),
      ],
    });

    playResolveTurnEvents(
      [
        { type: 'unitDamaged', unitId: 0x21, newHp: 0 },
        { type: 'unitDied', unitId: 0x21 },
      ],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map([[0x21, unitGraphics as never]]),
        bombGraphicsById: new Map(),
        softBlockGraphicsById: new Map(),
        onError: vi.fn(),
      }
    );

    // unitDamaged (offset 0, newHp<=0) draws the fire but schedules no auto-removal.
    delayedCallAt(0)();
    expect(unitGraphics.destroy).not.toHaveBeenCalled();

    // unitDied is scheduled at offset(0) + FIRE_DURATION_MS.
    delayedCallAt(FIRE_DURATION_MS)();
    expect(unitGraphics.destroy).toHaveBeenCalled();
  });
});

// Drains every scheduled delayedCall in insertion order, including ones appended by callbacks
// while draining (unitDied/softBlockDestroyed schedule nested removals). The index cursor keeps
// each callback firing exactly once.
function fireAllDelayedCalls(): void {
  let i = 0;
  while (i < mockScene.time.delayedCall.mock.calls.length) {
    const cb = mockScene.time.delayedCall.mock.calls[i]![1] as () => void;
    i++;
    cb();
  }
}

// Plays the event stream to completion, then asserts the graphics maps hold exactly the
// occupants the expected end-state says survived.
describe('playResolveTurnEvents — render fidelity oracle', () => {
  interface FidelityCase {
    name: string;
    initial: GameState;
    events: GameEvent[];
    // The occupants that must remain after playback (live units, surviving bombs/softBlocks).
    expected: GameState;
  }

  const grid = plainGrid(1, 5);
  const cases: FidelityCase[] = [
    {
      name: 'a bomb explosion that kills a unit and destroys a softBlock leaves all three maps empty',
      initial: makeState({
        grid,
        units: [makeUnit({ id: 0x21, position: { x: 1, y: 0 }, team: 2, hp: 1 })],
        bombs: [{ id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 3, countdown: 0 }],
        softBlocks: [makeSoftBlock({ id: 20, position: { x: 2, y: 0 } })],
      }),
      events: [
        {
          type: 'bombExploded',
          bombId: 1,
          affectedPositions: [
            { x: 1, y: 0 },
            { x: 2, y: 0 },
          ],
        },
        { type: 'unitDamaged', unitId: 0x21, newHp: 0 },
        { type: 'unitDied', unitId: 0x21 },
        { type: 'softBlockDestroyed', softBlockId: 20 },
      ],
      expected: makeState({
        grid,
        units: [makeUnit({ id: 0x21, position: { x: 1, y: 0 }, team: 2, hp: 0 })],
        bombs: [],
        softBlocks: [],
      }),
    },
    {
      name: 'a unit that only takes damage survives in the map while the spent bomb is removed',
      initial: makeState({
        grid,
        units: [makeUnit({ id: 0x21, position: { x: 1, y: 0 }, team: 2, hp: 2 })],
        bombs: [{ id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 3, countdown: 0 }],
      }),
      events: [
        { type: 'bombExploded', bombId: 1, affectedPositions: [{ x: 1, y: 0 }] },
        { type: 'unitDamaged', unitId: 0x21, newHp: 1 },
      ],
      expected: makeState({
        grid,
        units: [makeUnit({ id: 0x21, position: { x: 1, y: 0 }, team: 2, hp: 1 })],
        bombs: [],
      }),
    },
  ];

  it.each(cases)('$name', ({ initial, events, expected }) => {
    const unitGraphicsById = new Map(initial.units.map(u => [u.id, createMockGraphics() as never]));
    const bombGraphicsById = new Map<number, BombGraphics>(
      initial.bombs.map(b => [b.id, makeBombGraphics()])
    );
    const softBlockGraphicsById = new Map(
      initial.softBlocks.map(s => [s.id, createMockGraphics() as never])
    );

    const result = playResolveTurnEvents(events, {
      scene: mockScene as never,
      gameStateSnapshot: initial,
      unitGraphicsById,
      bombGraphicsById,
      softBlockGraphicsById,
      onError: vi.fn(),
    });
    expect(result.ok).toBe(true);

    fireAllDelayedCalls();

    expect(
      occupantsMatch(expected, unitGraphicsById, bombGraphicsById, softBlockGraphicsById)
    ).toBe(true);
  });

  it('discriminates: a leftover graphics entry that truth no longer has fails the oracle', () => {
    const initial = makeState({
      grid,
      bombs: [{ id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 1, countdown: 0 }],
    });
    const bombGraphicsById = new Map<number, BombGraphics>([[1, makeBombGraphics()]]);

    // Play NO events, so the bomb graphics linger — but expected end-state has no bombs.
    playResolveTurnEvents([], {
      scene: mockScene as never,
      gameStateSnapshot: initial,
      unitGraphicsById: new Map(),
      bombGraphicsById,
      softBlockGraphicsById: new Map(),
      onError: vi.fn(),
    });

    expect(occupantsMatch(makeState({ grid }), new Map(), bombGraphicsById, new Map())).toBe(false);
  });
});

describe('playResolveTurnEvents — done promise', () => {
  it('resolves once the longest scheduled effect (blast growth + lingering tail) fires', async () => {
    const state = baseState({
      grid: grid5x5(),
      bombs: [{ id: 1, ownerId: 0x11, position: { x: 0, y: 0 }, range: 1, countdown: 0 }],
    });

    const result = playResolveTurnEvents(
      [{ type: 'bombExploded', bombId: 1, affectedPositions: [{ x: 1, y: 0 }] }],
      {
        scene: mockScene as never,
        gameStateSnapshot: state,
        unitGraphicsById: new Map(),
        bombGraphicsById: new Map([[1, makeBombGraphics()]]),
        softBlockGraphicsById: new Map(),
        onError: vi.fn(),
      }
    );

    // 1 tile away => BLAST_SPEED_MS_PER_TILE growth + BLAST_DURATION_MS lingering tail.
    let settled = false;
    void result.done.then(() => {
      settled = true;
    });

    delayedCallAt(BLAST_SPEED_MS_PER_TILE + BLAST_DURATION_MS)();
    await Promise.resolve();
    expect(settled).toBe(true);
  });
});
