import { describe, it, expect } from 'vitest';
import { firstNonBaseFrame } from './spriteFrames';

function fakeScene(frames: Record<string, unknown>) {
  return {
    textures: { get: () => ({ frames }) },
  } as never;
}

describe('firstNonBaseFrame', () => {
  it('returns the one real frame name, skipping __BASE', () => {
    const scene = fakeScene({ __BASE: {}, 'Fighter #Blue.aseprite': {} });
    expect(firstNonBaseFrame(scene, 'unit_fighter_blue')).toBe('Fighter #Blue.aseprite');
  });

  it('throws when only __BASE exists', () => {
    const scene = fakeScene({ __BASE: {} });
    expect(() => firstNonBaseFrame(scene, 'broken')).toThrow(/no real frame/);
  });
});
