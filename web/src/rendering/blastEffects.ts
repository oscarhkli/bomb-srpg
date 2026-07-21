import type Phaser from 'phaser';
import type { Coordinate } from '../types/api';
import { TILE_SIZE, GAME_FONT_FAMILY } from '../constants';
import {
  DEPTH_BLAST,
  DEPTH_FIRE,
  FIRE_GLYPH,
  FIRE_SHAPE_SIZE,
  BLAST_ALPHA,
  BLAST_BEAM_WIDTH,
  BLAST_COLOR_OUTER,
  BLAST_COLOR_MID,
  BLAST_COLOR_INNER,
} from './constants';

export type CardinalDirection = 'N' | 'S' | 'E' | 'W';

function tileCenter(position: Coordinate): { cx: number; cy: number } {
  return {
    cx: position.x * TILE_SIZE + TILE_SIZE / 2,
    cy: position.y * TILE_SIZE + TILE_SIZE / 2,
  };
}

// A 🔥 glyph at the tile center — placeholder for a formal sprite, so kept minimal
export function drawFireShape(scene: Phaser.Scene, position: Coordinate): Phaser.GameObjects.Text {
  const { cx, cy } = tileCenter(position);

  const text = scene.add.text(cx, cy, FIRE_GLYPH, {
    fontFamily: GAME_FONT_FAMILY,
    fontSize: `${FIRE_SHAPE_SIZE}px`,
  });
  text.setOrigin(0.5);
  text.setDepth(DEPTH_FIRE);

  return text;
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

// Draws a pill-shaped (fully-rounded) rect; radius is clamped to half the segment length.
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
