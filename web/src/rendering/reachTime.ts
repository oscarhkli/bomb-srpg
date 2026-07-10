import type { Coordinate } from '../types/api';
import { BLAST_SPEED_MS_PER_TILE } from './constants';

// Manhattan distance — safe as a "cardinal distance" here since every affected tile in a
// bombExplodedEvent's affectedPositions is axis-aligned with the bomb (one of dx/dy is 0).
export function cardinalDistance(a: Coordinate, b: Coordinate): number {
  return Math.abs(a.x - b.x) + Math.abs(a.y - b.y);
}

export function reachTimeMs(from: Coordinate, to: Coordinate): number {
  return cardinalDistance(from, to) * BLAST_SPEED_MS_PER_TILE;
}
