import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { tweenConfigAt } from '../test/sceneHelpers';
import {
  FIRE_GLYPH,
  FIRE_SHAPE_SIZE,
  DEPTH_FIRE,
  DEPTH_BLAST,
  BLAST_ALPHA,
  BLAST_BEAM_WIDTH,
  BLAST_COLOR_INNER,
  BLAST_COLOR_MID,
  BLAST_COLOR_OUTER,
} from './constants';
import { TILE_SIZE } from '../constants';
import { drawFireShape, drawGrowingBeam } from './blastEffects';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('drawFireShape', () => {
  it('draws a 42px 🔥 glyph centered on the tile at DEPTH_FIRE', () => {
    const cx = 1 * TILE_SIZE + TILE_SIZE / 2;
    const cy = 2 * TILE_SIZE + TILE_SIZE / 2;

    drawFireShape(mockScene as never, { x: 1, y: 2 });

    expect(mockScene.add.text).toHaveBeenCalledWith(
      cx,
      cy,
      FIRE_GLYPH,
      expect.objectContaining({ fontSize: `${FIRE_SHAPE_SIZE}px` })
    );
    const text = mockScene.add.text.mock.results[0]!.value as ReturnType<typeof mockScene.add.text>;
    expect(text.setOrigin).toHaveBeenCalledWith(0.5);
    expect(text.setDepth).toHaveBeenCalledWith(DEPTH_FIRE);
  });
});

describe('drawGrowingBeam', () => {
  function tweenConfig(): {
    targets: { len: number };
    len: number;
    duration: number;
    onUpdate: () => void;
  } {
    return tweenConfigAt(0) as {
      targets: { len: number };
      len: number;
      duration: number;
      onUpdate: () => void;
    };
  }

  it('schedules a tween growing to the direction’s full pixel length over the given duration', () => {
    drawGrowingBeam(mockScene as never, { x: 0, y: 0 }, 'E', 3, 360);

    const cfg = tweenConfig();
    expect(cfg.targets.len).toBe(0);
    expect(cfg.len).toBe(3 * TILE_SIZE); // 144px
    expect(cfg.duration).toBe(360);
  });

  it('redraws only the innermost band while growth is within the first third of the length', () => {
    const g = drawGrowingBeam(
      mockScene as never,
      { x: 0, y: 0 },
      'E',
      3,
      360
    ) as unknown as ReturnType<typeof mockScene.add.graphics>;
    const cfg = tweenConfig();

    cfg.targets.len = 48; // exactly the first third of 144px
    cfg.onUpdate();

    expect(g.clear).toHaveBeenCalledTimes(1);
    expect(g.fillStyle).toHaveBeenCalledExactlyOnceWith(BLAST_COLOR_INNER, BLAST_ALPHA);
    // Pill-shaped (rounded rect), not a hard-edged rectangle — radius is half the beam width,
    // clamped to half the segment's own length so a short segment doesn't over-round.
    expect(g.fillRoundedRect).toHaveBeenCalledExactlyOnceWith(
      24,
      24 - BLAST_BEAM_WIDTH / 2,
      48,
      BLAST_BEAM_WIDTH,
      BLAST_BEAM_WIDTH / 2
    );
  });

  it('redraws all 3 gradient bands once growth reaches full length, oriented eastward from the origin', () => {
    const g = drawGrowingBeam(
      mockScene as never,
      { x: 0, y: 0 },
      'E',
      3,
      360
    ) as unknown as ReturnType<typeof mockScene.add.graphics>;
    const cfg = tweenConfig();

    cfg.targets.len = 144;
    cfg.onUpdate();

    expect(g.fillStyle).toHaveBeenNthCalledWith(1, BLAST_COLOR_INNER, BLAST_ALPHA);
    expect(g.fillRoundedRect).toHaveBeenNthCalledWith(
      1,
      24,
      24 - BLAST_BEAM_WIDTH / 2,
      48,
      BLAST_BEAM_WIDTH,
      BLAST_BEAM_WIDTH / 2
    );
    expect(g.fillStyle).toHaveBeenNthCalledWith(2, BLAST_COLOR_MID, BLAST_ALPHA);
    expect(g.fillRoundedRect).toHaveBeenNthCalledWith(
      2,
      24 + 48,
      24 - BLAST_BEAM_WIDTH / 2,
      48,
      BLAST_BEAM_WIDTH,
      BLAST_BEAM_WIDTH / 2
    );
    expect(g.fillStyle).toHaveBeenNthCalledWith(3, BLAST_COLOR_OUTER, BLAST_ALPHA);
    expect(g.fillRoundedRect).toHaveBeenNthCalledWith(
      3,
      24 + 96,
      24 - BLAST_BEAM_WIDTH / 2,
      48,
      BLAST_BEAM_WIDTH,
      BLAST_BEAM_WIDTH / 2
    );
  });

  it('clamps the pill radius to half the segment length when the growing tip is shorter than the beam width', () => {
    const g = drawGrowingBeam(
      mockScene as never,
      { x: 0, y: 0 },
      'E',
      3,
      360
    ) as unknown as ReturnType<typeof mockScene.add.graphics>;
    const cfg = tweenConfig();

    cfg.targets.len = 10; // shorter than BLAST_BEAM_WIDTH (32), so radius must clamp to 10/2=5
    cfg.onUpdate();

    expect(g.fillRoundedRect).toHaveBeenCalledExactlyOnceWith(
      24,
      24 - BLAST_BEAM_WIDTH / 2,
      10,
      BLAST_BEAM_WIDTH,
      5
    );
  });

  it('is set at DEPTH_BLAST', () => {
    const g = drawGrowingBeam(
      mockScene as never,
      { x: 0, y: 0 },
      'E',
      3,
      360
    ) as unknown as ReturnType<typeof mockScene.add.graphics>;
    expect(g.setDepth).toHaveBeenCalledWith(DEPTH_BLAST);
  });
});
