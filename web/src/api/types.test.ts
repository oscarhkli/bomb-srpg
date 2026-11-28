import { describe, expect, it } from 'vitest';
import { bombID, softBlockID, unitID } from '../test/ids';
import { makeBomb, makeState } from '../test/fixtures';
import {
  SYSTEM_OWNER_ID,
  type BombID,
  type CpuStatusResponse,
  type GameEvent,
  type GameState,
  type ResolveTurnResponse,
  type SoftBlockID,
  type TurnCommand,
  type UnitID,
} from './types';

const takesUnit = (id: UnitID) => id;
const takesBomb = (id: BombID) => id;
const takesSoftBlock = (id: SoftBlockID) => id;
const takesNumber = (n: number) => n;

function eventIDs(evt: GameEvent): number[] {
  switch (evt.type) {
    case 'unitMoved':
    case 'unitDamaged':
    case 'unitDied':
      return [takesUnit(evt.unitId)];
    case 'bombPlaced':
      return [takesUnit(evt.unitId), takesBomb(evt.bombId)];
    case 'bombCountdownUpdated':
    case 'bombExploded':
      return [takesBomb(evt.bombId)];
    case 'softBlockDestroyed':
      return [takesSoftBlock(evt.softBlockId)];
    case 'matchEnded':
      return [];
  }
}

function commandIDs(cmd: TurnCommand): number[] {
  return [takesUnit(cmd.unitId)];
}

function stateIDs(gs: GameState): number[] {
  return [
    ...gs.units.map(u => takesUnit(u.id)),
    ...gs.bombs.map(b => takesBomb(b.id)),
    ...gs.bombs.map(b => takesUnit(b.ownerId)),
    ...gs.softBlocks.map(s => takesSoftBlock(s.id)),
    ...gs.turnCommands.flatMap(commandIDs),
    ...gs.turnStartEvents.flatMap(eventIDs),
  ];
}

function planIDs(res: CpuStatusResponse | ResolveTurnResponse): number[] {
  return [...res.planGameEvents, ...res.resolveTurnGameEvents].flatMap(eventIDs);
}

describe('branded IDs', () => {
  it('pass through the sinks and read from every nested path', () => {
    const evt: GameEvent = {
      type: 'bombPlaced',
      unitId: unitID(1),
      bombId: bombID(2),
      position: { x: 0, y: 0 },
      range: 2,
      countdown: 3,
    };
    const gs = makeState({ turnStartEvents: [evt] });
    expect(stateIDs(gs)).toEqual([1, 2]);
    expect(planIDs({ planGameEvents: [evt], resolveTurnGameEvents: [evt] })).toEqual([1, 2, 1, 2]);
    expect(takesSoftBlock(softBlockID(3))).toBe(3);
  });

  it('accepts a branded ID where a number is expected, and the system owner as a unit', () => {
    expect(takesNumber(unitID(4))).toBe(4);
    expect(takesUnit(SYSTEM_OWNER_ID)).toBe(0);
  });

  it('rejects another kind of ID or a bare number', () => {
    const unit = unitID(1);
    const bomb = bombID(1);
    const softBlock = softBlockID(1);
    // @ts-expect-error a bomb ID is not a unit ID
    takesUnit(bomb);
    // @ts-expect-error a soft block ID is not a bomb ID
    takesBomb(softBlock);
    // @ts-expect-error a unit ID is not a soft block ID
    takesSoftBlock(unit);
    // @ts-expect-error a bare number is not a unit ID
    takesUnit(1);
    const bombWithBareOwner = makeBomb({
      // @ts-expect-error a bare number is not an owner unit ID
      ownerId: 1,
    });
    expect(bombWithBareOwner.ownerId).toBe(1);
  });
});
