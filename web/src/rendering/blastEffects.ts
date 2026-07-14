import Phaser from 'phaser';
import type { Coordinate } from '../types/api';
import { TILE_SIZE } from '../constants';
import {
  DEPTH_BLAST,
  DEPTH_FIRE,
  FIRE_ALPHA,
  FIRE_SHAPE_SIZE,
  BLAST_ALPHA,
  BLAST_BEAM_WIDTH,
  BLAST_COLOR_OUTER,
  BLAST_COLOR_MID,
  BLAST_COLOR_INNER,
} from './constants';

const FIRE_COLOR = 0xff6600;

export type CardinalDirection = 'N' | 'S' | 'E' | 'W';

function tileCenter(position: Coordinate): { cx: number; cy: number } {
  return {
    cx: position.x * TILE_SIZE + TILE_SIZE / 2,
    cy: position.y * TILE_SIZE + TILE_SIZE / 2,
  };
}

// A mild flame shape (two overlapping triangles) — polish is explicitly out of scope here.
export function drawFireShape(
  scene: Phaser.Scene,
  position: Coordinate
): Phaser.GameObjects.Graphics {
  const { cx, cy } = tileCenter(position);
  const r = FIRE_SHAPE_SIZE / 2;

  const g = scene.add.graphics();
  g.setDepth(DEPTH_FIRE);
  g.fillStyle(FIRE_COLOR, FIRE_ALPHA);
  g.fillPoints(
    [
      new Phaser.Math.Vector2(cx, cy - r),
      new Phaser.Math.Vector2(cx - r * 0.6, cy + r * 0.6),
      new Phaser.Math.Vector2(cx + r * 0.6, cy + r * 0.6),
    ],
    true
  );
  g.fillPoints(
    [
      new Phaser.Math.Vector2(cx, cy - r * 0.5),
      new Phaser.Math.Vector2(cx - r * 0.35, cy + r),
      new Phaser.Math.Vector2(cx + r * 0.35, cy + r),
    ],
    true
  );

  return g;
}

interface BandSegment {
  from: number;
  to: number;
  color: number;
}

// Splits [0, totalLength] into inner/mid/outer thirds (nearest-to-bomb = inner/hottest,
// farthest tip = outer), clipped to how far the beam has grown (currentLen) so far.
function bandSegments(totalLength: number, currentLen: number): BandSegment[] {
  const thirdLen = totalLength / 3;
  const boundaries = [0, thirdLen, thirdLen * 2, totalLength];
  const colors = [BLAST_COLOR_INNER, BLAST_COLOR_MID, BLAST_COLOR_OUTER];

  const segments: BandSegment[] = [];
  for (let i = 0; i < 3; i++) {
    const from = boundaries[i]!;
    const to = Math.min(boundaries[i + 1]!, currentLen);
    if (to > from) {
      segments.push({ from, to, color: colors[i]! });
    }
  }
  return segments;
}

// Draws a pill shape (fully-rounded rect) rather than a hard-edged rectangle, so the beam's
// leading tip reads as a rounded "head" rather than a flat blocky end. Radius is clamped to
// half the segment's own length so a very short (just-starting-to-grow) segment doesn't
// request a radius larger than its own geometry allows.
function drawSegmentRect(
  g: Phaser.GameObjects.Graphics,
  cx: number,
  cy: number,
  direction: CardinalDirection,
  from: number,
  to: number
): void {
  const length = to - from;
  const radius = Math.min(BLAST_BEAM_WIDTH / 2, length / 2);
  switch (direction) {
    case 'E':
      g.fillRoundedRect(cx + from, cy - BLAST_BEAM_WIDTH / 2, length, BLAST_BEAM_WIDTH, radius);
      break;
    case 'W':
      g.fillRoundedRect(cx - to, cy - BLAST_BEAM_WIDTH / 2, length, BLAST_BEAM_WIDTH, radius);
      break;
    case 'S':
      g.fillRoundedRect(cx - BLAST_BEAM_WIDTH / 2, cy + from, BLAST_BEAM_WIDTH, length, radius);
      break;
    case 'N':
      g.fillRoundedRect(cx - BLAST_BEAM_WIDTH / 2, cy - to, BLAST_BEAM_WIDTH, length, radius);
      break;
  }
}

// Renders one cardinal ray of a bomb's blast as a beam that elongates outward over time —
// a single tween drives redraws rather than per-tile flashes.
export function drawGrowingBeam(
  scene: Phaser.Scene,
  origin: Coordinate,
  direction: CardinalDirection,
  maxDistTiles: number,
  durationMs: number
): Phaser.GameObjects.Graphics {
  const { cx, cy } = tileCenter(origin);
  const totalLengthPx = maxDistTiles * TILE_SIZE;

  const g = scene.add.graphics();
  g.setDepth(DEPTH_BLAST);

  const state = { len: 0 };
  const redraw = (): void => {
    g.clear();
    for (const seg of bandSegments(totalLengthPx, state.len)) {
      g.fillStyle(seg.color, BLAST_ALPHA);
      drawSegmentRect(g, cx, cy, direction, seg.from, seg.to);
    }
  };

  scene.tweens.add({
    targets: state,
    len: totalLengthPx,
    duration: durationMs,
    onUpdate: redraw,
  });

  return g;
}
