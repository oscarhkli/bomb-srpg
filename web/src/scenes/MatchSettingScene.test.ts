import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { firstText } from '../test/sceneHelpers';
import MatchSettingScene from './MatchSettingScene';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MatchSettingScene', () => {
  it('renders a centered "Match Settings" title', () => {
    const scene = new MatchSettingScene();

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
