import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { fadeOutTargets } from './gameObjectUtils';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('fadeOutTargets', () => {
  it('tweens the given targets to alpha 0 and invokes onComplete when the tween finishes', () => {
    const onComplete = vi.fn();
    const targets = [{ alpha: 1 }, { alpha: 1 }];

    fadeOutTargets(mockScene as never, targets, 200, onComplete);

    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ targets, alpha: 0, duration: 200 })
    );
    const { onComplete: tweenOnComplete } = mockScene.tweens.add.mock.calls[0]![0] as {
      onComplete: () => void;
    };
    expect(onComplete).not.toHaveBeenCalled();
    tweenOnComplete();
    expect(onComplete).toHaveBeenCalledOnce();
  });
});
