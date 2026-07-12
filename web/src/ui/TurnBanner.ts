import type Phaser from 'phaser';
import {
  DEPTH_TURN_BANNER,
  GAME_FONT_FAMILY,
  TEAM_COLORS,
  TURN_BANNER_FADE_MS,
  TURN_BANNER_FONT_SIZE,
  TURN_BANNER_HEIGHT,
  TURN_BANNER_HOLD_MS,
  TURN_BANNER_TEXT_COLOR,
} from '../constants';
import { colorToCss } from './gameObjectUtils';

// Full-width turn-transition banner: fades in, holds, fades out, then destroys itself.
// play() resolves once the whole sequence (and the destroy) is complete, so MatchScene's
// beginTurn() can await it before re-enabling interactions.
export default class TurnBanner {
  constructor(private readonly scene: Phaser.Scene) {}

  play(activeTeam: number): Promise<void> {
    return new Promise(resolve => {
      const { width, height } = this.scene.cameras.main;
      const y = height / 2 - TURN_BANNER_HEIGHT / 2;

      const bg = this.scene.add.graphics();
      bg.setDepth(DEPTH_TURN_BANNER);
      bg.setScrollFactor(0);
      bg.fillStyle(TEAM_COLORS[activeTeam] ?? 0xffffff);
      bg.fillRect(0, y, width, TURN_BANNER_HEIGHT);
      bg.alpha = 0;

      const text = this.scene.add.text(
        width / 2,
        y + TURN_BANNER_HEIGHT / 2,
        `Player ${activeTeam}'s Turn`,
        {
          fontFamily: GAME_FONT_FAMILY,
          fontSize: `${TURN_BANNER_FONT_SIZE}px`,
          color: colorToCss(TURN_BANNER_TEXT_COLOR),
        }
      );
      text.setOrigin(0.5);
      text.setDepth(DEPTH_TURN_BANNER);
      text.setScrollFactor(0);
      text.alpha = 0;

      const targets = [bg, text];

      const destroyAndResolve = (): void => {
        bg.destroy();
        text.destroy();
        resolve();
      };

      const fadeOut = (): void => {
        this.scene.tweens.add({
          targets,
          alpha: 0,
          duration: TURN_BANNER_FADE_MS,
          onComplete: destroyAndResolve,
        });
      };

      const hold = (): void => {
        this.scene.time.delayedCall(TURN_BANNER_HOLD_MS, fadeOut);
      };

      this.scene.tweens.add({
        targets,
        alpha: 1,
        duration: TURN_BANNER_FADE_MS,
        onComplete: hold,
      });
    });
  }
}
