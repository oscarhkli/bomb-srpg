import Phaser from 'phaser';
import {
  FADE_MS,
  GAME_FONT_FAMILY,
  TITLE_TOP_MARGIN,
  TITLE_FONT_SIZE,
  TITLE_GAME_MODE_FONT_SIZE,
  TITLE_COPYRIGHT_BOTTOM_MARGIN,
  TITLE_COPYRIGHT_FONT_SIZE,
  TITLE_HOVER_BOMB_GAP,
} from '../constants';

// Entry scene: game title, game mode selection, and copyright line.
export default class TitleScene extends Phaser.Scene {
  // Guards against re-entrant option clicks once a transition has started.
  private isTransitioning = false;

  constructor() {
    super('TitleScene');
  }

  // Registers the game font browser-wide before any Text rasterizes; canvas text never
  // triggers a lazy stylesheet font fetch, so it must be loaded explicitly.
  preload(): void {
    this.load.font('Roboto', 'fonts/roboto-400.woff2', 'woff2');
  }

  create(): void {
    this.isTransitioning = false;
    this.cameras.main.fadeIn(FADE_MS);
    this.renderTitle();
    this.renderGameModeSelectionPanel();
    this.renderCopyrightText();
  }

  private renderTitle(): void {
    const centerX = this.cameras.main.width / 2;
    const line1 = this.add.text(centerX, TITLE_TOP_MARGIN, 'Bomb', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${TITLE_FONT_SIZE}px`,
    });
    line1.setOrigin(0.5, 0);

    // Line 2's "T" starts at the x of line 1's "m" — measured via a throwaway "Bo" text.
    const prefix = this.add.text(0, 0, 'Bo', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${TITLE_FONT_SIZE}px`,
    });
    const indent = prefix.width;
    prefix.destroy();

    const line2 = this.add.text(
      centerX - line1.width / 2 + indent,
      TITLE_TOP_MARGIN + line1.height,
      'Tactics',
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${TITLE_FONT_SIZE}px`,
      }
    );
    line2.setOrigin(0, 0);
  }

  private renderGameModeSelectionPanel(): void {
    const { width, height } = this.cameras.main;
    const option = this.add.text(width / 2, height / 2, 'Start Game', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${TITLE_GAME_MODE_FONT_SIZE}px`,
    });
    option.setOrigin(0.5);
    option.setInteractive({ useHandCursor: true });

    let hoverBomb: Phaser.GameObjects.Text | undefined;
    option.on('pointerover', () => {
      hoverBomb?.destroy();
      hoverBomb = this.add.text(
        option.x - option.width / 2 - TITLE_HOVER_BOMB_GAP,
        option.y,
        '💣',
        {
          fontFamily: GAME_FONT_FAMILY,
          fontSize: `${TITLE_GAME_MODE_FONT_SIZE}px`,
        }
      );
      hoverBomb.setOrigin(1, 0.5);
    });
    option.on('pointerout', () => {
      hoverBomb?.destroy();
      hoverBomb = undefined;
    });
    option.on('pointerdown', () => {
      if (this.isTransitioning) {
        return;
      }
      this.isTransitioning = true;
      option.disableInteractive();
      this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
      this.cameras.main.once('camerafadeoutcomplete', () => {
        // {} (not omitted): Phaser retains the previous scene data when start() gets no
        // data argument, which would resurrect the last match's settings.
        this.scene.start('MatchSettingsScene', {});
      });
    });
  }

  private renderCopyrightText(): void {
    const { width, height } = this.cameras.main;
    const text = this.add.text(
      width / 2,
      height - TITLE_COPYRIGHT_BOTTOM_MARGIN,
      `© ${new Date().getFullYear()} Oscar oscarhkli.com`,
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${TITLE_COPYRIGHT_FONT_SIZE}px`,
      }
    );
    text.setOrigin(0.5, 1);
  }
}
