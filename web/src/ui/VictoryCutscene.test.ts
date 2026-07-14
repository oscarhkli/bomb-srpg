import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  allGraphics,
  allTexts,
  firstGraphics,
  graphicsAt,
  pointerDownOf,
  tweenConfigAt,
} from '../test/sceneHelpers';
import {
  TEAM_COLORS,
  TEAM_COLOR_FALLBACK,
  VICTORY_BUTTON_DELAY_MS,
  DEPTH_VICTORY_CUTSCENE,
} from '../constants';
import VictoryCutscene from './VictoryCutscene';

beforeEach(() => {
  vi.clearAllMocks();
});

// Drives the fade-in -> button-delay chain, mirroring TurnBanner.test.ts's
// completeBannerSequence() helper.
function completeFadeIn(): void {
  const fadeInCall = tweenConfigAt(0) as { onComplete?: () => void };
  fadeInCall.onComplete?.();
}

function completeFadeInAndButtonDelay(): void {
  completeFadeIn();
  const delayedCall = mockScene.time.delayedCall.mock.calls.find(
    c => c[0] === VICTORY_BUTTON_DELAY_MS
  );
  (delayedCall?.[1] as () => void)();
}

describe('VictoryCutscene', () => {
  it.each([
    [1, TEAM_COLORS[1]],
    [2, TEAM_COLORS[2]],
  ])('fills the banner with TEAM_COLORS[%i] for a non-draw result', (winnerTeamId, color) => {
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(winnerTeamId, { onRematch: vi.fn(), onReturnToSettings: vi.fn() });

    // graphicsAt(0) is the full-canvas scrim, graphicsAt(1) is the banner itself.
    expect(graphicsAt(1).fillStyle).toHaveBeenCalledWith(color);
  });

  it('fills the banner with TEAM_COLOR_FALLBACK and shows "Draw Game" for winnerTeamId -1', () => {
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(-1, { onRematch: vi.fn(), onReturnToSettings: vi.fn() });

    expect(graphicsAt(1).fillStyle).toHaveBeenCalledWith(TEAM_COLOR_FALLBACK);
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Draw Game',
      expect.objectContaining({ fontSize: '48px' })
    );
  });

  it('renders "Winner..." shifted 5% left of center and "Player {X}!" fully centered, for a non-draw result', () => {
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(2, { onRematch: vi.fn(), onReturnToSettings: vi.fn() });

    // Camera width is 1280 in the test mock, so 5% left of center (640) is 576.
    expect(mockScene.add.text).toHaveBeenCalledWith(
      576,
      expect.any(Number),
      'Winner...',
      expect.objectContaining({ fontSize: '36px' })
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      640,
      expect.any(Number),
      'Player 2!',
      expect.objectContaining({ fontSize: '48px' })
    );
    for (const text of allTexts()) {
      expect(text.setOrigin).toHaveBeenCalledWith(0.5);
    }
  });

  it('covers the full canvas with a dim scrim pinned to the camera at DEPTH_VICTORY_CUTSCENE', () => {
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(1, { onRematch: vi.fn(), onReturnToSettings: vi.fn() });

    const scrim = firstGraphics();
    expect(scrim.fillRect).toHaveBeenCalledWith(0, 0, 1280, 720);
    expect(scrim.setScrollFactor).toHaveBeenCalledWith(0);
    expect(scrim.setDepth).toHaveBeenCalledWith(DEPTH_VICTORY_CUTSCENE);
  });

  it('does not render the Rematch/Return buttons before the fade-in and 2s delay complete', () => {
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(1, { onRematch: vi.fn(), onReturnToSettings: vi.fn() });

    expect(mockScene.add.text).not.toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Rematch',
      expect.objectContaining({})
    );
  });

  it('renders Rematch and Return to Match Settings buttons 2s after the fade-in completes', () => {
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(1, { onRematch: vi.fn(), onReturnToSettings: vi.fn() });
    completeFadeInAndButtonDelay();

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Rematch',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Return to Match Settings',
      expect.objectContaining({})
    );
  });

  it('invokes onRematch when the Rematch button is clicked', () => {
    const onRematch = vi.fn();
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(1, { onRematch, onReturnToSettings: vi.fn() });
    completeFadeInAndButtonDelay();

    // graphicsAt(0)=scrim, (1)=banner (text lines are add.text, not add.graphics), so the
    // Rematch button's own Graphics is the 3rd graphics call.
    const rematchButtonGraphics = graphicsAt(2);
    pointerDownOf(rematchButtonGraphics)();

    expect(onRematch).toHaveBeenCalledOnce();
  });

  it('invokes onReturnToSettings when the Return to Match Settings button is clicked', () => {
    const onReturnToSettings = vi.fn();
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(1, { onRematch: vi.fn(), onReturnToSettings });
    completeFadeInAndButtonDelay();

    // graphicsAt(2) is the Rematch button's Graphics; the Return button's is the next one.
    const returnButtonGraphics = graphicsAt(3);
    pointerDownOf(returnButtonGraphics)();

    expect(onReturnToSettings).toHaveBeenCalledOnce();
  });

  it('never fades the banner back out or destroys any object on its own', () => {
    const cutscene = new VictoryCutscene(mockScene as never);

    cutscene.play(1, { onRematch: vi.fn(), onReturnToSettings: vi.fn() });
    completeFadeInAndButtonDelay();

    expect(mockScene.tweens.add).toHaveBeenCalledTimes(1);
    for (const g of allGraphics()) {
      expect(g.destroy).not.toHaveBeenCalled();
    }
  });
});
