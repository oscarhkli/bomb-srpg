import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { firstText } from '../test/sceneHelpers';
import MatchSettingsScene from './MatchSettingsScene';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MatchSettingsScene', () => {
  it('renders a centered "Match Settings" title', () => {
    const scene = new MatchSettingsScene();

    scene.create();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      640,
      360,
      'Match Settings',
      expect.objectContaining({})
    );
    expect(firstText().setOrigin).toHaveBeenCalledWith(0.5);
  });
});
