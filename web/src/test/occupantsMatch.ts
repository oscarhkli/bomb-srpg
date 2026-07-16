// Test-only render-fidelity oracle. Asserts the invariant "the occupant graphics maps contain
// exactly the occupants that server truth says exist" — every live unit, every bomb, every
// softBlock has a graphics entry, and nothing extra. This is the assertion for resolveTurnPlayer /
// render-fidelity tests: it catches the client apply-code bugs that a runtime diff cannot justify
// guarding against (see spec007). It lives test-side and is never imported by production code.
import type Phaser from 'phaser';
import type { GameState } from '../types/api';
import type { BombGraphics } from '../rendering/resolveTurnPlayer';

export function occupantsMatch(
  state: GameState,
  unitGraphicsById: Map<number, Phaser.GameObjects.Graphics>,
  bombGraphicsById: Map<number, BombGraphics>,
  softBlockGraphicsById: Map<number, Phaser.GameObjects.Graphics>
): boolean {
  const liveUnits = state.units.filter(u => u.hp > 0);
  if (liveUnits.length !== unitGraphicsById.size) {
    return false;
  }
  if (!liveUnits.every(u => unitGraphicsById.has(u.id))) {
    return false;
  }

  if (state.bombs.length !== bombGraphicsById.size) {
    return false;
  }
  if (!state.bombs.every(b => bombGraphicsById.has(b.id))) {
    return false;
  }

  if (state.softBlocks.length !== softBlockGraphicsById.size) {
    return false;
  }
  if (!state.softBlocks.every(s => softBlockGraphicsById.has(s.id))) {
    return false;
  }

  return true;
}
