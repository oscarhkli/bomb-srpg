import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  textCalls,
  textByContent,
  fireTextPointerEvent,
  fireCameraFadeOutComplete,
  flush,
} from '../test/sceneHelpers';
import { createMatchRoom, initRoom, createMatch } from '../engine/api';
import TitleScene from './TitleScene';

vi.mock('../engine/api');

function bootScene(): TitleScene {
  const scene = new TitleScene();
  scene.create();
  return scene;
}

function openSubmenu(): void {
  fireTextPointerEvent(textByContent('Start Game'), 'pointerdown');
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('TitleScene — static content', () => {
  it.each([
    ['Title line 1', 'Bomb'],
    ['Title line 2', 'Tactics'],
    ['GameModeSelectionPanel option', 'Start Game'],
    ['CopyrightText with current year', `© ${new Date().getFullYear()} Oscar oscarhkli.com`],
  ])('renders %s', (_label, content) => {
    bootScene();

    expect(textCalls().some(c => c[2] === content)).toBe(true);
  });

  it('preloads the self-hosted game font before first paint', () => {
    new TitleScene().preload();

    expect(mockScene.load.font).toHaveBeenCalledWith('Roboto', 'fonts/roboto-400.woff2', 'woff2');
  });

  it('fades in on create(), completing the fadeTransition pair when re-entered', () => {
    bootScene();

    expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
  });
});

describe('TitleScene — game mode option hover', () => {
  it('renders a 💣 left of the option on pointerover and removes it on pointerout', () => {
    bootScene();
    const option = textByContent('Start Game');

    fireTextPointerEvent(option, 'pointerover');
    expect(textCalls().some(c => c[2] === '💣')).toBe(true);

    const bomb = textByContent('💣');
    fireTextPointerEvent(option, 'pointerout');
    expect(bomb.destroy).toHaveBeenCalled();
  });

  it('does not orphan a 💣 when pointerover fires twice without pointerout', () => {
    bootScene();
    const option = textByContent('Start Game');

    fireTextPointerEvent(option, 'pointerover');
    const firstBomb = textByContent('💣');
    fireTextPointerEvent(option, 'pointerover');

    expect(firstBomb.destroy).toHaveBeenCalled();
  });

  it('destroys a hover 💣 left over from clicking an option without a pointerout, before the submenu opens', () => {
    bootScene();
    const option = textByContent('Start Game');

    fireTextPointerEvent(option, 'pointerover');
    const bomb = textByContent('💣');
    fireTextPointerEvent(option, 'pointerdown'); // no pointerout fired first

    expect(bomb.destroy).toHaveBeenCalled();
  });
});

describe('TitleScene — Start Game submenu (AC 1, 2)', () => {
  it('shows Story Mode / Battle Mode / Back stacked vertically where Start Game was, sharing its font (AC 1)', () => {
    bootScene();
    const rootOption = textByContent('Start Game');
    const rootCall = textCalls().find(c => c[2] === 'Start Game')!;

    openSubmenu();

    expect(rootOption.setVisible).toHaveBeenCalledWith(false);
    const ys: number[] = [];
    for (const label of ['Story Mode', 'Battle Mode', 'Back']) {
      const option = textByContent(label);
      expect(option.setVisible).toHaveBeenCalledWith(true);
      const optionCall = textCalls().find(c => c[2] === label)!;
      expect(optionCall[0]).toBe(rootCall[0]); // same x as root
      expect(optionCall[3]).toMatchObject({
        fontFamily: (rootCall[3] as { fontFamily: string }).fontFamily,
      });
      ys.push(optionCall[1]);
    }
    // Distinct, ascending y — a vertical list, not 3 lines stacked on the exact same spot.
    expect(new Set(ys).size).toBe(3);
    expect(ys).toEqual([...ys].sort((a, b) => a - b));
  });

  it('restores Start Game and hides the submenu when Back is clicked (AC 2)', () => {
    bootScene();
    openSubmenu();
    const rootOption = textByContent('Start Game');
    vi.mocked(rootOption.setVisible).mockClear();

    fireTextPointerEvent(textByContent('Back'), 'pointerdown');

    expect(rootOption.setVisible).toHaveBeenCalledWith(true);
    expect(textByContent('Story Mode').setVisible).toHaveBeenCalledWith(false);
  });
});

describe('TitleScene — Story Mode (AC 3, 4)', () => {
  it('creates a match with the Prologue GameCfg and starts MatchScene with roomId + playerTokens (AC 3)', async () => {
    vi.mocked(createMatchRoom).mockResolvedValue({ id: 'room-xyz' });
    vi.mocked(createMatch).mockResolvedValue({ success: true, playerTokens: ['t1', 't2'] });
    bootScene();
    openSubmenu();

    fireTextPointerEvent(textByContent('Story Mode'), 'pointerdown');
    await flush();
    fireCameraFadeOutComplete();
    await flush();

    expect(initRoom).toHaveBeenCalledWith('room-xyz');
    const cfg = vi.mocked(createMatch).mock.calls[0]![0].gameCfg;
    expect(cfg.vsCpu).toBe(true);
    expect(cfg.p2Slots).toEqual([{ archetype: 'Prologue', role: 'Boss' }]);
    expect(mockScene.scene.start).toHaveBeenCalledWith('MatchScene', {
      roomId: 'room-xyz',
      playerTokens: ['t1', 't2'],
    });
  });

  it('reports the error and stays on TitleScene when createMatchRoom fails (AC 4)', async () => {
    vi.mocked(createMatchRoom).mockRejectedValue(new Error('network error'));
    bootScene();
    openSubmenu();

    fireTextPointerEvent(textByContent('Story Mode'), 'pointerdown');
    await flush();
    fireCameraFadeOutComplete();
    await flush();

    expect(mockScene.scene.start).not.toHaveBeenCalled();
    expect(textCalls().some(c => typeof c[2] === 'string' && c[2].includes('match'))).toBe(true);
  });
});

describe('TitleScene — Battle Mode', () => {
  it('fadeTransitions to MatchSettingsScene without carrying settings on click', () => {
    bootScene();
    openSubmenu();

    fireTextPointerEvent(textByContent('Battle Mode'), 'pointerdown');
    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);

    fireCameraFadeOutComplete();
    expect(mockScene.scene.start).toHaveBeenCalledWith('MatchSettingsScene', {});
  });

  it('ignores further clicks once the transition has started', () => {
    bootScene();
    openSubmenu();
    const option = textByContent('Battle Mode');

    fireTextPointerEvent(option, 'pointerdown');
    fireTextPointerEvent(option, 'pointerdown');
    fireCameraFadeOutComplete();

    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledTimes(1);
    expect(mockScene.scene.start).toHaveBeenCalledTimes(1);
  });
});
