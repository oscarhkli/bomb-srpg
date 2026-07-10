import type Phaser from 'phaser';
import type { Coordinate, GameEvent, GameState, TurnCommand } from '../types/api';
import type { BombGraphics } from './resolveTurnPlayer';

// Client/server state-reconciliation helpers. Pure (no Phaser scene needed) so they can be
// unit-tested in isolation; used by MatchScene's turn-command and post-resolve sanity checks.

export type AppliedTurnResult =
  | { type: 'move'; unitId: number; to: Coordinate }
  | { type: 'placeBomb'; bombId: number; position: Coordinate };

// Extracts what the server actually reported happened (event data), not what the client
// originally requested — a future engine change (push/swap) could make these differ.
export function extractAppliedTarget(
  cmd: TurnCommand,
  events: GameEvent[]
): AppliedTurnResult | undefined {
  if (cmd.type === 'move') {
    const e = events.find(ev => ev.type === 'unitMoved');
    if (e?.unitId === undefined || !e.to) {
      return undefined;
    }
    return { type: 'move', unitId: e.unitId, to: e.to };
  }
  const e = events.find(ev => ev.type === 'bombPlaced');
  if (e?.bombId === undefined || !e.position) {
    return undefined;
  }
  return { type: 'placeBomb', bombId: e.bombId, position: e.position };
}

// Spot-checks only the entity a turn command touched: a `move` never changes *who* exists
// (only position), so a full existence sweep would be a no-op for it — this checks the one
// thing the command was actually supposed to change. Compares against the server-reported
// event data (`AppliedTurnResult`), not the originally requested command.
export function turnCommandTargetMatches(
  freshState: GameState,
  result: AppliedTurnResult
): boolean {
  if (result.type === 'move') {
    const unit = freshState.units.find(u => u.id === result.unitId);
    return unit?.position.x === result.to.x && unit?.position.y === result.to.y;
  }
  const bomb = freshState.bombs.find(b => b.id === result.bombId);
  return bomb?.position.x === result.position.x && bomb?.position.y === result.position.y;
}

export function occupantsMatch(
  freshState: GameState,
  unitGraphicsById: Map<number, Phaser.GameObjects.Graphics>,
  bombGraphicsById: Map<number, BombGraphics>,
  softBlockGraphicsById: Map<number, Phaser.GameObjects.Graphics>
): boolean {
  const liveUnits = freshState.units.filter(u => u.hp > 0);
  if (liveUnits.length !== unitGraphicsById.size) {
    return false;
  }
  if (!liveUnits.every(u => unitGraphicsById.has(u.id))) {
    return false;
  }

  if (freshState.bombs.length !== bombGraphicsById.size) {
    return false;
  }
  if (!freshState.bombs.every(b => bombGraphicsById.has(b.id))) {
    return false;
  }

  if (freshState.softBlocks.length !== softBlockGraphicsById.size) {
    return false;
  }
  if (!freshState.softBlocks.every(s => softBlockGraphicsById.has(s.id))) {
    return false;
  }

  return true;
}
