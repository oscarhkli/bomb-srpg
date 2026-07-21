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

export interface ResolveTurnPlayerDeps {
  scene: Phaser.Scene;
  gameStateSnapshot: GameState;
  unitGraphicsById: Map<number, Phaser.GameObjects.Graphics>;
  bombGraphicsById: Map<number, BombGraphics>;
  softBlockGraphicsById: Map<number, Phaser.GameObjects.Graphics>;
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
        const { bombId, affectedPositions } = event;
        if (bombId === undefined || affectedPositions === undefined) {
          return 'bombExploded event is missing bombId/affectedPositions';
        }
        const bomb = snapshot.bombs.find(b => b.id === bombId);
        if (!bomb) {
          return `bombExploded event references unknown bombId ${bombId}`;
        }
        for (const p of affectedPositions) {
          const row = snapshot.grid[p.y];
          if (!row?.[p.x]) {
            return `bombExploded event has an out-of-bounds affected position (${p.x}, ${p.y})`;
          }
        }
        break;
      }
      case 'unitDamaged': {
        const { unitId, newHp } = event;
        if (unitId === undefined || newHp === undefined) {
          return 'unitDamaged event is missing unitId/newHp';
        }
        if (!Number.isInteger(newHp) || newHp < 0) {
          return `unitDamaged event has an invalid newHp ${newHp}`;
        }
        if (!snapshot.units.some(u => u.id === unitId)) {
          return `unitDamaged event references unknown unitId ${unitId}`;
        }
        break;
      }
      case 'unitDied': {
        const { unitId } = event;
        if (unitId === undefined) {
          return 'unitDied event is missing unitId';
        }
        if (!snapshot.units.some(u => u.id === unitId)) {
          return `unitDied event references unknown unitId ${unitId}`;
        }
        break;
      }
      case 'softBlockDestroyed': {
        const { softBlockId } = event;
        if (softBlockId === undefined) {
          return 'softBlockDestroyed event is missing softBlockId';
        }
        if (!snapshot.softBlocks.some(s => s.id === softBlockId)) {
          return `softBlockDestroyed event references unknown softBlockId ${softBlockId}`;
        }
        break;
      }
      default:
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
  deps: ResolveTurnPlayerDeps
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
    const { bombId, affectedPositions } = event;
    const position = deps.gameStateSnapshot.bombs.find(b => b.id === bombId)!.position;
    const offset = causerOffsetFor(position, explodedList);
    explodedList.push({ bombId: bombId!, position, affectedPositions: affectedPositions!, offset });
  }

  for (const info of explodedList) {
    const { bombId, position, affectedPositions, offset } = info;
    const byDirection = directionMaxDistances(position, affectedPositions);

    deps.scene.time.delayedCall(offset, () => {
      const bg = deps.bombGraphicsById.get(bombId);
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
      const { unitId, newHp } = event;
      const position = deps.gameStateSnapshot.units.find(u => u.id === unitId)!.position;
      const offset = causerOffsetFor(position, explodedList);
      deps.scene.time.delayedCall(offset, () => {
        const fire = drawFireShape(deps.scene, position);
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
      const { unitId } = event;
      const position = deps.gameStateSnapshot.units.find(u => u.id === unitId)!.position;
      const offset = causerOffsetFor(position, explodedList);
      deps.scene.time.delayedCall(offset + FIRE_DURATION_MS, () => {
        deps.unitGraphicsById.get(unitId!)?.destroy();
        deps.unitGraphicsById.delete(unitId!);
        fireByUnitId.get(unitId!)?.destroy();
        fireByUnitId.delete(unitId!);
      });
      endTimes.push(offset + FIRE_DURATION_MS);
    } else if (event.type === 'softBlockDestroyed') {
      const { softBlockId } = event;
      const position = deps.gameStateSnapshot.softBlocks.find(s => s.id === softBlockId)!.position;
      const offset = causerOffsetFor(position, explodedList);
      deps.scene.time.delayedCall(offset, () => {
        const fire = drawFireShape(deps.scene, position);
        deps.scene.time.delayedCall(FIRE_DURATION_MS, () => {
          fire.destroy();
          deps.softBlockGraphicsById.get(softBlockId!)?.destroy();
          deps.softBlockGraphicsById.delete(softBlockId!);
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
