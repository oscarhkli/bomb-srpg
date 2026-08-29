import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { flush, fireCameraFadeOutComplete } from '../test/sceneHelpers';
import { makeCfg } from '../test/fixtures';
import { createMatchRoom, initRoom, createMatch } from '../engine/api';
import ErrorPanel from '../ui/ErrorPanel';
import { startMatch } from './startMatch';

vi.mock('../engine/api');

beforeEach(() => {
  vi.clearAllMocks();
});

function callStartMatch(overrides: { isTransitioning?: boolean } = {}): {
  errorPanel: ErrorPanel;
  setTransitioning: ReturnType<typeof vi.fn>;
} {
  const errorPanel = new ErrorPanel(mockScene as never);
  const setTransitioning = vi.fn();
  startMatch({
    scene: mockScene as never,
    gameCfg: makeCfg(),
    isTransitioning: () => overrides.isTransitioning ?? false,
    setTransitioning,
    generation: () => 0,
    errorPanel,
  });
  return { errorPanel, setTransitioning };
}

describe('startMatch — success path', () => {
  it('creates the room and match, then starts MatchScene with roomId + playerTokens', async () => {
    vi.mocked(createMatchRoom).mockResolvedValue({ id: 'room-xyz' });
    vi.mocked(createMatch).mockResolvedValue({ success: true, playerTokens: ['t1', 't2'] });

    callStartMatch();
    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);

    await flush();
    fireCameraFadeOutComplete();
    await flush();

    expect(createMatchRoom).toHaveBeenCalled();
    expect(initRoom).toHaveBeenCalledWith('room-xyz');
    expect(mockScene.scene.start).toHaveBeenCalledWith('MatchScene', {
      roomId: 'room-xyz',
      playerTokens: ['t1', 't2'],
    });
  });
});

describe('startMatch — re-entrancy guard', () => {
  it('does nothing when isTransitioning() already reports true', () => {
    callStartMatch({ isTransitioning: true });

    expect(mockScene.cameras.main.fadeOut).not.toHaveBeenCalled();
  });
});

describe('startMatch — failure path', () => {
  it('fades back in, reports the error, and resets isTransitioning on createMatchRoom failure', async () => {
    vi.mocked(createMatchRoom).mockRejectedValue(new Error('network error'));

    const { setTransitioning } = callStartMatch();
    await flush();
    fireCameraFadeOutComplete();
    await flush();

    expect(mockScene.scene.start).not.toHaveBeenCalled();
    expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
    expect(setTransitioning).toHaveBeenCalledWith(false);
  });
});
