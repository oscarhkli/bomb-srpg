import type { Coordinate } from '../api/types';
import { BLAST_SPEED_MS_PER_TILE } from './constants';

// Manhattan distance; blast tiles are axis-aligned with the bomb, so this is the ray length.
export function cardinalDistance(a: Coordinate, b: Coordinate): number {
  return Math.abs(a.x - b.x) + Math.abs(a.y - b.y);
}

export function reachTimeMs(from: Coordinate, to: Coordinate): number {
  return cardinalDistance(from, to) * BLAST_SPEED_MS_PER_TILE;
}
