import type Phaser from 'phaser';
import {
  DEPTH_ERROR_PANEL,
  ERROR_PANEL_BG_ALPHA,
  ERROR_PANEL_BG_COLOR,
  ERROR_PANEL_HEIGHT,
  ERROR_PANEL_PADDING,
  ERROR_PANEL_WIDTH,
  ERROR_PANEL_X,
  ERROR_PANEL_Y,
} from '../constants';

// A camera-pinned panel that stacks error messages. The background is created lazily on the
// first show() and torn down on clear(); messages advance by their actual (word-wrapped)
// rendered height so long messages don't overlap the next one.
export default class ErrorPanel {
  private bg: Phaser.GameObjects.Graphics | undefined;
  private texts: Phaser.GameObjects.Text[] = [];
  private nextY = ERROR_PANEL_Y + ERROR_PANEL_PADDING;

  constructor(private readonly scene: Phaser.Scene) {}

  show(message: string): void {
    if (!this.bg) {
      const bg = this.scene.add.graphics();
      bg.setDepth(DEPTH_ERROR_PANEL);
      bg.setScrollFactor(0);
      bg.fillStyle(ERROR_PANEL_BG_COLOR, ERROR_PANEL_BG_ALPHA);
      bg.fillRect(ERROR_PANEL_X, ERROR_PANEL_Y, ERROR_PANEL_WIDTH, ERROR_PANEL_HEIGHT);
      this.bg = bg;
    }

    const text = this.scene.add.text(ERROR_PANEL_X + ERROR_PANEL_PADDING, this.nextY, message, {
      wordWrap: { width: ERROR_PANEL_WIDTH - ERROR_PANEL_PADDING * 2 },
    });
    text.setDepth(DEPTH_ERROR_PANEL);
    text.setScrollFactor(0);
    this.texts.push(text);
    this.nextY += text.height + ERROR_PANEL_PADDING;
  }

  clear(): void {
    this.texts.forEach(t => t.destroy());
    this.texts = [];
    this.bg?.destroy();
    this.bg = undefined;
    this.nextY = ERROR_PANEL_Y + ERROR_PANEL_PADDING;
  }
}
