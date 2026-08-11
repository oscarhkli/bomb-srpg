// Test-only oracle: asserts the occupant graphics maps contain exactly the occupants that
// server truth says exist, nothing missing or extra.
import type Phaser from 'phaser';
import type { GameState } from '../types/api';
import type { BombGraphics } from '../rendering/resolveTurnPlayer';

export function occupantsMatch(
  state: GameState,
  unitSpritesById: Map<number, Phaser.GameObjects.Sprite>,
  bombGraphicsById: Map<number, BombGraphics>,
  softBlockSpritesById: Map<number, Phaser.GameObjects.Sprite>
): boolean {
  const liveUnits = state.units.filter(u => u.hp > 0);
  if (liveUnits.length !== unitSpritesById.size) {
    return false;
  }
  if (!liveUnits.every(u => unitSpritesById.has(u.id))) {
    return false;
  }

  if (state.bombs.length !== bombGraphicsById.size) {
    return false;
  }
  if (!state.bombs.every(b => bombGraphicsById.has(b.id))) {
    return false;
  }

  if (state.softBlocks.length !== softBlockSpritesById.size) {
    return false;
  }
  if (!state.softBlocks.every(s => softBlockSpritesById.has(s.id))) {
    return false;
  }

  return true;
}
