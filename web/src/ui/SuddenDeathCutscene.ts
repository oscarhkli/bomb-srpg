import type Phaser from 'phaser';
import {
  DEPTH_SUDDEN_DEATH_OVERLAY,
  SUDDEN_DEATH_BOMB_DROP_DELAY_MS,
  SUDDEN_DEATH_CUTSCENE_DURATION_MS,
  SUDDEN_DEATH_PULSE_HALF_MS,
  SUDDEN_DEATH_PULSE_PEAK_ALPHA,
  SUDDEN_DEATH_COLOR,
} from '../constants';
import type { GameEvent } from '../types/api';

// Sudden-death turn-transition effect: a full-canvas pulsing overlay, with bombs dropping from
// the sky partway through. play() resolves after the LATER of the pulse duration and the last
// bomb landing, so MatchScene's beginTurn() can await the whole sequence before continuing.
export default class SuddenDeathCutscene {
  constructor(private readonly scene: Phaser.Scene) {}

  play(
    bombPlacedEvents: GameEvent[],
    dropBomb: (event: GameEvent) => Promise<void>
  ): Promise<void> {
    const { width, height } = this.scene.cameras.main;

    const overlay = this.scene.add.graphics();
    overlay.setDepth(DEPTH_SUDDEN_DEATH_OVERLAY);
    overlay.setScrollFactor(0);
    overlay.fillStyle(SUDDEN_DEATH_COLOR);
    overlay.fillRect(0, 0, width, height);
    overlay.alpha = 0;

    this.scene.tweens.add({
      targets: overlay,
      alpha: SUDDEN_DEATH_PULSE_PEAK_ALPHA,
      duration: SUDDEN_DEATH_PULSE_HALF_MS,
      yoyo: true,
      repeat: -1,
    });

    const pulseDone = new Promise<void>(resolve => {
      this.scene.time.delayedCall(SUDDEN_DEATH_CUTSCENE_DURATION_MS, () => {
        overlay.destroy();
        resolve();
      });
    });

    const bombsDone = new Promise<void>(resolve => {
      this.scene.time.delayedCall(SUDDEN_DEATH_BOMB_DROP_DELAY_MS, () => {
        Promise.all(bombPlacedEvents.map(event => dropBomb(event)))
          .then(() => resolve())
          .catch(() => resolve());
      });
    });

    return Promise.all([pulseDone, bombsDone]).then(() => undefined);
  }
}
