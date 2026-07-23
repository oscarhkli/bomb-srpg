import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  textCalls,
  textByContent,
  fireTextPointerEvent,
  fireCameraFadeOutComplete,
} from '../test/sceneHelpers';
import TitleScene from './TitleScene';

function bootScene(): TitleScene {
  const scene = new TitleScene();
  scene.create();
  return scene;
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
  ])('renders %s (AC 1, 8)', (_label, content) => {
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

describe('TitleScene — game mode option hover (AC 2)', () => {
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

  it('disables interactivity once the transition has started', () => {
    bootScene();
    const option = textByContent('Start Game');

    fireTextPointerEvent(option, 'pointerdown');

    expect(option.disableInteractive).toHaveBeenCalled();
  });
});

describe('TitleScene — starting a game (AC 3, 7)', () => {
  it('fadeTransitions to MatchSettingsScene without carrying settings on click', () => {
    bootScene();

    fireTextPointerEvent(textByContent('Start Game'), 'pointerdown');
    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);

    fireCameraFadeOutComplete();
    expect(mockScene.scene.start).toHaveBeenCalledWith('MatchSettingsScene', {});
  });

  it('ignores further clicks once the transition has started (AC 4)', () => {
    bootScene();
    const option = textByContent('Start Game');

    fireTextPointerEvent(option, 'pointerdown');
    fireTextPointerEvent(option, 'pointerdown');
    fireCameraFadeOutComplete();

    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledTimes(1);
    expect(mockScene.scene.start).toHaveBeenCalledTimes(1);
  });
});
