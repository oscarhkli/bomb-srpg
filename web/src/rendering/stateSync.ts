import type Phaser from 'phaser';
import type { GameState, Tile } from '../types/api';
import type { BombGraphics } from './resolveTurnPlayer';

// Client/server state-reconciliation helpers. Pure (no Phaser scene needed) so they can be
// unit-tested in isolation; used by MatchScene's optimistic-update and post-resolve sanity checks.

export function cloneGrid(grid: Tile[][]): Tile[][] {
  return grid.map(row => row.map(tile => ({ ...tile })));
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

export function gridsEqual(a: Tile[][], b: Tile[][]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  for (let y = 0; y < a.length; y++) {
    const rowA = a[y];
    const rowB = b[y];
    if (!rowA || rowA.length !== rowB?.length) {
      return false;
    }
    for (let x = 0; x < rowA.length; x++) {
      const tileA = rowA[x];
      const tileB = rowB[x];
      if (
        !tileA ||
        tileA.type !== tileB?.type ||
        tileA.occupantType !== tileB.occupantType ||
        tileA.occupantId !== tileB.occupantId
      ) {
        return false;
      }
    }
  }
  return true;
}
