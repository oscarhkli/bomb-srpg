import { describe, expect, it } from 'vitest';
import Phaser from 'phaser';
import { computeZoom } from './scale';

describe('computeZoom', () => {
  it('returns NO_ZOOM when width is below the breakpoint', () => {
    expect(computeZoom(1359, 848)).toBe(Phaser.Scale.NO_ZOOM);
  });

  it('returns NO_ZOOM when height is below the breakpoint', () => {
    expect(computeZoom(1360, 847)).toBe(Phaser.Scale.NO_ZOOM);
  });

  it('returns ZOOM_2X exactly at both breakpoints', () => {
    expect(computeZoom(1360, 848)).toBe(Phaser.Scale.ZOOM_2X);
  });

  it('returns ZOOM_2X well above both breakpoints', () => {
    expect(computeZoom(1920, 1080)).toBe(Phaser.Scale.ZOOM_2X);
  });

  it('returns NO_ZOOM well below both breakpoints', () => {
    expect(computeZoom(800, 600)).toBe(Phaser.Scale.NO_ZOOM);
  });
});
