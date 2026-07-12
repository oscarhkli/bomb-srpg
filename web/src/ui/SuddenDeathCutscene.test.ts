import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  firstGraphics as overlayGraphics,
  tweenConfigAt,
  fireDelayedCall,
} from '../test/sceneHelpers';
import { makeBombPlacedEvent } from '../test/fixtures';
import {
  SUDDEN_DEATH_COLOR,
  DEPTH_SUDDEN_DEATH_OVERLAY,
  SUDDEN_DEATH_CUTSCENE_DURATION_MS,
  SUDDEN_DEATH_BOMB_DROP_DELAY_MS,
  SUDDEN_DEATH_PULSE_HALF_MS,
} from '../constants';
import SuddenDeathCutscene from './SuddenDeathCutscene';

beforeEach(() => {
  vi.clearAllMocks();
});

function pulseCall(): { duration?: number; yoyo?: boolean; repeat?: number } {
  return tweenConfigAt(0) as { duration?: number; yoyo?: boolean; repeat?: number };
}

describe('SuddenDeathCutscene', () => {
  it('renders a full-canvas overlay filled with SUDDEN_DEATH_COLOR at DEPTH_SUDDEN_DEATH_OVERLAY', () => {
    const cutscene = new SuddenDeathCutscene(mockScene as never);

    void cutscene.play([], () => Promise.resolve());

    expect(overlayGraphics().fillStyle).toHaveBeenCalledWith(SUDDEN_DEATH_COLOR);
    expect(overlayGraphics().setScrollFactor).toHaveBeenCalledWith(0);
    expect(overlayGraphics().setDepth).toHaveBeenCalledWith(DEPTH_SUDDEN_DEATH_OVERLAY);
  });

  it('pulses the overlay alpha via a yoyo, repeating tween for the pulse-half duration', () => {
    const cutscene = new SuddenDeathCutscene(mockScene as never);

    void cutscene.play([], () => Promise.resolve());

    expect(pulseCall()).toEqual(
      expect.objectContaining({
        duration: SUDDEN_DEATH_PULSE_HALF_MS,
        yoyo: true,
        repeat: -1,
      })
    );
  });

  it('shows and pulses the overlay even with zero bombPlaced events, and never calls dropBomb', () => {
    const dropBomb = vi.fn().mockResolvedValue(undefined);
    const cutscene = new SuddenDeathCutscene(mockScene as never);

    void cutscene.play([], dropBomb);
    fireDelayedCall(SUDDEN_DEATH_BOMB_DROP_DELAY_MS);

    expect(overlayGraphics().fillStyle).toHaveBeenCalledWith(SUDDEN_DEATH_COLOR);
    expect(dropBomb).not.toHaveBeenCalled();
  });

  it('invokes dropBomb once per bombPlaced event, only after the drop-delay elapses', () => {
    const dropBomb = vi.fn().mockResolvedValue(undefined);
    const events = [makeBombPlacedEvent({ bombId: 1 }), makeBombPlacedEvent({ bombId: 2 })];
    const cutscene = new SuddenDeathCutscene(mockScene as never);

    void cutscene.play(events, dropBomb);
    expect(dropBomb).not.toHaveBeenCalled();

    fireDelayedCall(SUDDEN_DEATH_BOMB_DROP_DELAY_MS);

    expect(dropBomb).toHaveBeenCalledTimes(2);
    expect(dropBomb).toHaveBeenNthCalledWith(1, events[0]);
    expect(dropBomb).toHaveBeenNthCalledWith(2, events[1]);
  });

  it('resolves play() only after BOTH the pulse duration and all dropBomb promises settle', async () => {
    let resolveDrop: () => void = () => undefined;
    const dropBomb = vi.fn().mockReturnValue(
      new Promise<void>(resolve => {
        resolveDrop = resolve;
      })
    );
    const events = [makeBombPlacedEvent()];
    const cutscene = new SuddenDeathCutscene(mockScene as never);

    let resolved = false;
    const playPromise = cutscene.play(events, dropBomb).then(() => {
      resolved = true;
    });

    fireDelayedCall(SUDDEN_DEATH_BOMB_DROP_DELAY_MS); // triggers dropBomb, still pending
    fireDelayedCall(SUDDEN_DEATH_CUTSCENE_DURATION_MS); // pulse duration elapses
    await Promise.resolve();
    await Promise.resolve();
    expect(resolved).toBe(false); // dropBomb promise still pending

    resolveDrop();
    await playPromise;
    expect(resolved).toBe(true);
  });

  it('resolves play() only after the pulse duration when dropBomb resolves first', async () => {
    const dropBomb = vi.fn().mockResolvedValue(undefined);
    const events = [makeBombPlacedEvent()];
    const cutscene = new SuddenDeathCutscene(mockScene as never);

    let resolved = false;
    const playPromise = cutscene.play(events, dropBomb).then(() => {
      resolved = true;
    });

    fireDelayedCall(SUDDEN_DEATH_BOMB_DROP_DELAY_MS);
    await Promise.resolve();
    await Promise.resolve();
    expect(resolved).toBe(false); // pulse duration hasn't elapsed yet

    fireDelayedCall(SUDDEN_DEATH_CUTSCENE_DURATION_MS);
    await playPromise;
    expect(resolved).toBe(true);
  });

  it('destroys the overlay once the pulse duration elapses', () => {
    const cutscene = new SuddenDeathCutscene(mockScene as never);

    void cutscene.play([], () => Promise.resolve());
    const g = overlayGraphics();
    fireDelayedCall(SUDDEN_DEATH_CUTSCENE_DURATION_MS);

    expect(g.destroy).toHaveBeenCalled();
  });
});
