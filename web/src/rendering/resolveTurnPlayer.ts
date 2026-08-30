import type Phaser from 'phaser';
import type { Coordinate, GameEvent, GameState } from '../types/api';
import {
  BOMB_COUNTDOWN_ZERO_COLOR,
  BOMB_COUNTDOWN_TEXT_COLOR,
  BLAST_SPEED_MS_PER_TILE,
  BLAST_DURATION_MS,
  FIRE_DURATION_MS,
} from './constants';
import { cardinalDistance, reachTimeMs } from './reachTime';
import { drawGrowingBeam, drawFireShape, type CardinalDirection } from './blastEffects';
import { colorToCss } from '../ui/gameObjectUtils';

export interface BombGraphics {
  container: Phaser.GameObjects.Container;
  countdownText: Phaser.GameObjects.Text;
}

export interface ResolveTurnPlayerOptions {
  scene: Phaser.Scene;
  gameStateSnapshot: GameState;
  unitSpritesById: Map<number, Phaser.GameObjects.Sprite>;
  bombGraphicsById: Map<number, BombGraphics>;
  softBlockSpritesById: Map<number, Phaser.GameObjects.Sprite>;
  onError: (message: string) => void;
}

export interface PlayResult {
  ok: boolean;
  done: Promise<void>;
}

function renderBombCountdownText(text: Phaser.GameObjects.Text, countdown: number): void {
  if (countdown === 0) {
    text.setText('!');
    text.setColor(colorToCss(BOMB_COUNTDOWN_ZERO_COLOR));
  } else {
    text.setText(String(countdown));
    text.setColor(colorToCss(BOMB_COUNTDOWN_TEXT_COLOR));
  }
}

function inBounds(snapshot: GameState, p: Coordinate): boolean {
  return snapshot.grid[p.y]?.[p.x] !== undefined;
}

function validate(events: GameEvent[], snapshot: GameState): string | null {
  for (const event of events) {
    switch (event.type) {
      case 'bombCountdownUpdated': {
        const { bombId, countdown } = event;
        if (bombId === undefined || countdown === undefined) {
          return 'bombCountdownUpdated event is missing bombId/countdown';
        }
        if (!Number.isInteger(countdown) || countdown < 0) {
          return `bombCountdownUpdated event has an invalid countdown ${countdown}`;
        }
        if (!snapshot.bombs.some(b => b.id === bombId)) {
          return `bombCountdownUpdated event references unknown bombId ${bombId}`;
        }
        break;
      }
      case 'bombExploded': {
        const { bombId, position, affectedPositions } = event;
        if (bombId === undefined || !position || affectedPositions === undefined) {
          return 'bombExploded event is missing bombId/position/affectedPositions';
        }
        if (!inBounds(snapshot, position)) {
          return `bombExploded event has an out-of-bounds position (${position.x}, ${position.y})`;
        }
        for (const p of affectedPositions) {
          if (!inBounds(snapshot, p)) {
            return `bombExploded event has an out-of-bounds affected position (${p.x}, ${p.y})`;
          }
        }
        break;
      }
      case 'unitDamaged': {
        const { unitId, newHp, position } = event;
        if (unitId === undefined || newHp === undefined || !position) {
          return 'unitDamaged event is missing unitId/newHp/position';
        }
        if (!Number.isInteger(newHp) || newHp < 0) {
          return `unitDamaged event has an invalid newHp ${newHp}`;
        }
        if (!snapshot.units.some(u => u.id === unitId)) {
          return `unitDamaged event references unknown unitId ${unitId}`;
        }
        if (!inBounds(snapshot, position)) {
          return `unitDamaged event has an out-of-bounds position (${position.x}, ${position.y})`;
        }
        break;
      }
      case 'unitDied': {
        const { unitId, position } = event;
        if (unitId === undefined || !position) {
          return 'unitDied event is missing unitId/position';
        }
        if (!snapshot.units.some(u => u.id === unitId)) {
          return `unitDied event references unknown unitId ${unitId}`;
        }
        if (!inBounds(snapshot, position)) {
          return `unitDied event has an out-of-bounds position (${position.x}, ${position.y})`;
        }
        break;
      }
      case 'softBlockDestroyed': {
        const { softBlockId, position } = event;
        if (softBlockId === undefined || !position) {
          return 'softBlockDestroyed event is missing softBlockId/position';
        }
        if (!snapshot.softBlocks.some(s => s.id === softBlockId)) {
          return `softBlockDestroyed event references unknown softBlockId ${softBlockId}`;
        }
        if (!inBounds(snapshot, position)) {
          return `softBlockDestroyed event has an out-of-bounds position (${position.x}, ${position.y})`;
        }
        break;
      }
      case 'matchEnded':
        // MatchScene handles this one, not us.
        break;
      default:
        if (import.meta.env.DEV) {
          console.warn(`playResolveTurnEvents received an unhandled event type: ${event.type}`);
        }
        break;
    }
  }
  return null;
}

interface ExplodedInfo {
  bombId: number;
  position: Coordinate;
  affectedPositions: Coordinate[];
  offset: number;
}

// Chain-reaction/occupant causer: whichever earlier bombExplodedEvent's blast reaches
// `position` soonest (smallest resulting offset), not necessarily the earliest in event order.
function causerOffsetFor(position: Coordinate, exploded: ExplodedInfo[]): number {
  let best: number | undefined;
  for (const e of exploded) {
    if (e.affectedPositions.some(p => p.x === position.x && p.y === position.y)) {
      const candidate = e.offset + reachTimeMs(e.position, position);
      if (best === undefined || candidate < best) {
        best = candidate;
      }
    }
  }
  return best ?? 0;
}

function directionOf(bombPos: Coordinate, tile: Coordinate): CardinalDirection | null {
  if (tile.x === bombPos.x && tile.y < bombPos.y) {
    return 'N';
  }
  if (tile.x === bombPos.x && tile.y > bombPos.y) {
    return 'S';
  }
  if (tile.y === bombPos.y && tile.x > bombPos.x) {
    return 'E';
  }
  if (tile.y === bombPos.y && tile.x < bombPos.x) {
    return 'W';
  }
  return null;
}

function directionMaxDistances(
  bombPos: Coordinate,
  affectedPositions: Coordinate[]
): Map<CardinalDirection, number> {
  const byDirection = new Map<CardinalDirection, number>();
  for (const tile of affectedPositions) {
    const dir = directionOf(bombPos, tile);
    if (!dir) {
      continue;
    }
    const dist = cardinalDistance(bombPos, tile);
    byDirection.set(dir, Math.max(byDirection.get(dir) ?? 0, dist));
  }
  return byDirection;
}

export function playResolveTurnEvents(
  events: GameEvent[],
  deps: ResolveTurnPlayerOptions
): PlayResult {
  const validationError = validate(events, deps.gameStateSnapshot);
  if (validationError !== null) {
    deps.onError(validationError);
    return { ok: false, done: Promise.resolve() };
  }

  const endTimes: number[] = [0];

  for (const event of events) {
    if (event.type === 'bombCountdownUpdated') {
      const { bombId, countdown } = event;
      deps.scene.time.delayedCall(0, () => {
        const bg = deps.bombGraphicsById.get(bombId!);
        if (bg) {
          renderBombCountdownText(bg.countdownText, countdown!);
        }
      });
    }
  }

  const explodedList: ExplodedInfo[] = [];
  for (const event of events) {
    if (event.type !== 'bombExploded') {
      continue;
    }
    const { bombId, position, affectedPositions } = event;
    const offset = causerOffsetFor(position!, explodedList);
    explodedList.push({
      bombId: bombId!,
      position: position!,
      affectedPositions: affectedPositions!,
      offset,
    });
  }

  for (const info of explodedList) {
    const { bombId, position, affectedPositions, offset } = info;
    const byDirection = directionMaxDistances(position, affectedPositions);

    deps.scene.time.delayedCall(offset, () => {
      const bg = deps.bombGraphicsById.get(bombId);
      if (!bg) {
        // Tolerated: a bombId with no live graphics (e.g. a bug upstream) still renders the
        // explosion at the event's own position — there's nothing to clean up for it locally.
        console.warn(`bombExploded event references a bomb with no live graphics: ${bombId}`);
      }
      bg?.container.destroy();
      deps.bombGraphicsById.delete(bombId);

      for (const [dir, maxDist] of byDirection) {
        const durationMs = maxDist * BLAST_SPEED_MS_PER_TILE;
        const beam = drawGrowingBeam(deps.scene, position, dir, maxDist, durationMs);
        deps.scene.time.delayedCall(durationMs + BLAST_DURATION_MS, () => beam.destroy());
      }
    });

    for (const maxDist of byDirection.values()) {
      endTimes.push(offset + maxDist * BLAST_SPEED_MS_PER_TILE + BLAST_DURATION_MS);
    }
  }

  const fireByUnitId = new Map<number, Phaser.GameObjects.Text>();

  for (const event of events) {
    if (event.type === 'unitDamaged') {
      const { unitId, newHp, position } = event;
      const offset = causerOffsetFor(position!, explodedList);
      deps.scene.time.delayedCall(offset, () => {
        const fire = drawFireShape(deps.scene, position!);
        fireByUnitId.set(unitId!, fire);
        if (newHp! > 0) {
          deps.scene.time.delayedCall(FIRE_DURATION_MS, () => {
            fire.destroy();
            fireByUnitId.delete(unitId!);
          });
        }
      });
      if (newHp! > 0) {
        endTimes.push(offset + FIRE_DURATION_MS);
      }
    } else if (event.type === 'unitDied') {
      const { unitId, position } = event;
      const offset = causerOffsetFor(position!, explodedList);
      deps.scene.time.delayedCall(offset + FIRE_DURATION_MS, () => {
        deps.unitSpritesById.get(unitId!)?.destroy();
        deps.unitSpritesById.delete(unitId!);
        fireByUnitId.get(unitId!)?.destroy();
        fireByUnitId.delete(unitId!);
      });
      endTimes.push(offset + FIRE_DURATION_MS);
    } else if (event.type === 'softBlockDestroyed') {
      const { softBlockId, position } = event;
      const offset = causerOffsetFor(position!, explodedList);
      deps.scene.time.delayedCall(offset, () => {
        const fire = drawFireShape(deps.scene, position!);
        deps.scene.time.delayedCall(FIRE_DURATION_MS, () => {
          fire.destroy();
          deps.softBlockSpritesById.get(softBlockId!)?.destroy();
          deps.softBlockSpritesById.delete(softBlockId!);
        });
      });
      endTimes.push(offset + FIRE_DURATION_MS);
    }
  }

  const maxEndTime = Math.max(...endTimes);
  const done = new Promise<void>(resolve => {
    deps.scene.time.delayedCall(maxEndTime, resolve);
  });

  return { ok: true, done };
}
