import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { firstGraphics as bannerGraphics, firstText as bannerText } from '../test/sceneHelpers';
import {
  TEAM_COLORS,
  TURN_BANNER_FADE_MS,
  TURN_BANNER_HOLD_MS,
  DEPTH_TURN_BANNER,
} from '../constants';
import TurnBanner from './TurnBanner';

beforeEach(() => {
  vi.clearAllMocks();
});

// Drives the fade-in -> hold -> fade-out -> destroy chain by manually invoking each scheduled
// callback in order, since tweens.add / time.delayedCall are inert vi.fn() mocks.
function completeBannerSequence(): void {
  const fadeInCall = mockScene.tweens.add.mock.calls[0]![0] as { onComplete?: () => void };
  fadeInCall.onComplete?.();

  const holdCall = mockScene.time.delayedCall.mock.calls[0]!;
  (holdCall[1] as () => void)();

  const fadeOutCall = mockScene.tweens.add.mock.calls[1]![0] as { onComplete?: () => void };
  fadeOutCall.onComplete?.();
}

describe('TurnBanner', () => {
  it.each([
    [1, TEAM_COLORS[1]],
    [2, TEAM_COLORS[2]],
  ])('fills the banner with TEAM_COLORS[%i]', (activeTeam, color) => {
    const banner = new TurnBanner(mockScene as never);

    void banner.play(activeTeam);

    expect(bannerGraphics().fillStyle).toHaveBeenCalledWith(color);
  });

  it("renders 'Player {X}'s Turn' centered in the banner", () => {
    const banner = new TurnBanner(mockScene as never);

    void banner.play(2);

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      "Player 2's Turn",
      expect.objectContaining({})
    );
    expect(bannerText().setOrigin).toHaveBeenCalledWith(0.5);
  });

  it('pins the banner and its text to the camera viewport (scrollFactor 0) at DEPTH_TURN_BANNER', () => {
    const banner = new TurnBanner(mockScene as never);

    void banner.play(1);

    expect(bannerGraphics().setScrollFactor).toHaveBeenCalledWith(0);
    expect(bannerGraphics().setDepth).toHaveBeenCalledWith(DEPTH_TURN_BANNER);
    expect(bannerText().setScrollFactor).toHaveBeenCalledWith(0);
    expect(bannerText().setDepth).toHaveBeenCalledWith(DEPTH_TURN_BANNER);
  });

  it('schedules fade-in, hold, and fade-out with the configured durations', () => {
    const banner = new TurnBanner(mockScene as never);

    void banner.play(1);

    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ duration: TURN_BANNER_FADE_MS, alpha: 1 })
    );

    const fadeInCall = mockScene.tweens.add.mock.calls[0]![0] as { onComplete?: () => void };
    fadeInCall.onComplete?.();

    expect(mockScene.time.delayedCall).toHaveBeenCalledWith(
      TURN_BANNER_HOLD_MS,
      expect.any(Function)
    );

    const holdCall = mockScene.time.delayedCall.mock.calls[0]!;
    (holdCall[1] as () => void)();

    expect(mockScene.tweens.add).toHaveBeenCalledWith(
      expect.objectContaining({ duration: TURN_BANNER_FADE_MS, alpha: 0 })
    );
  });

  it('destroys the banner and resolves play() once the full fade sequence completes', async () => {
    const banner = new TurnBanner(mockScene as never);

    const playPromise = banner.play(1);
    const g = bannerGraphics();
    const t = bannerText();
    completeBannerSequence();
    await playPromise;

    expect(g.destroy).toHaveBeenCalled();
    expect(t.destroy).toHaveBeenCalled();
  });
});
