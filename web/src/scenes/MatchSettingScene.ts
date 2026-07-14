import Phaser from 'phaser';
import { GAME_FONT_FAMILY } from '../constants';

// Stub scene: a rough landing page for scene entry. A concrete version will replace it later.
export default class MatchSettingScene extends Phaser.Scene {
  constructor() {
    super('MatchSettingScene');
  }

  create(): void {
    const { width, height } = this.cameras.main;
    const text = this.add.text(width / 2, height / 2, 'Match Settings', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: '32px',
      color: '#ffffff',
    });
    text.setOrigin(0.5);
  }
}
