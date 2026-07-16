import type Phaser from 'phaser';
import {
  CONFIRM_DIALOG_DIM_ALPHA,
  CONFIRM_DIALOG_DIM_COLOR,
  DEPTH_VICTORY_CUTSCENE,
  FADE_MS,
  GAME_FONT_FAMILY,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_FILL_COLOR,
  LIFECYCLE_BUTTON_HEIGHT_SMALL,
  LIFECYCLE_BUTTON_WIDTH,
  TEAM_COLOR_FALLBACK,
  TEAM_COLORS,
  TURN_BANNER_HEIGHT,
  TURN_BANNER_TEXT_COLOR,
  VICTORY_BUTTON_DELAY_MS,
  VICTORY_BUTTON_GAP,
  VICTORY_SUBTITLE_FONT_SIZE,
  VICTORY_SUBTITLE_OFFSET_Y,
  VICTORY_SUBTITLE_X_SHIFT_RATIO,
  VICTORY_TITLE_FONT_SIZE,
  VICTORY_TITLE_OFFSET_Y,
} from '../constants';
import { colorToCss, createFilledRect, fadeInTargets } from './gameObjectUtils';
import { drawPillButton, verticalButtonY } from './pillButton';

export interface VictoryCutsceneCallbacks {
  onRematch: () => void;
  onReturnToSettings: () => void;
}

const BUTTON_STYLE = {
  fillColor: PANEL_BUTTON_FILL_COLOR,
  fillAlpha: PANEL_BUTTON_FILL_ALPHA,
  borderColor: PANEL_BUTTON_BORDER_COLOR,
  borderWidth: PANEL_BUTTON_BORDER_WIDTH,
};

// Terminal full-canvas overlay shown once a match ends: a dim scrim + a banner announcing the
// winner (or draw), followed 2s later by Rematch/Return-to-Settings buttons. Unlike TurnBanner
// or SuddenDeathCutscene, this never fades out or destroys itself — it's the last thing shown in
// this MatchScene instance until the player picks a button (which tears the whole scene down).
export default class VictoryCutscene {
  constructor(private readonly scene: Phaser.Scene) {}

  play(winnerTeamId: number, callbacks: VictoryCutsceneCallbacks): void {
    const { width, height } = this.scene.cameras.main;
    const isDraw = winnerTeamId === -1;
    const bannerY = height / 2 - TURN_BANNER_HEIGHT / 2;

    const scrim = this.scene.add.graphics();
    scrim.setDepth(DEPTH_VICTORY_CUTSCENE);
    scrim.setScrollFactor(0);
    scrim.fillStyle(CONFIRM_DIALOG_DIM_COLOR, CONFIRM_DIALOG_DIM_ALPHA);
    scrim.fillRect(0, 0, width, height);

    const banner = createFilledRect(
      this.scene,
      0,
      bannerY,
      width,
      TURN_BANNER_HEIGHT,
      TEAM_COLORS[winnerTeamId] ?? TEAM_COLOR_FALLBACK,
      DEPTH_VICTORY_CUTSCENE
    );

    const texts = isDraw
      ? [this.addBannerText(width / 2, height / 2, 'Draw Game', VICTORY_TITLE_FONT_SIZE)]
      : [
          this.addBannerText(
            width / 2 - width * VICTORY_SUBTITLE_X_SHIFT_RATIO,
            bannerY + TURN_BANNER_HEIGHT / 2 + VICTORY_SUBTITLE_OFFSET_Y,
            'Winner...',
            VICTORY_SUBTITLE_FONT_SIZE
          ),
          this.addBannerText(
            width / 2,
            bannerY + TURN_BANNER_HEIGHT / 2 + VICTORY_TITLE_OFFSET_Y,
            `Player ${winnerTeamId}!`,
            VICTORY_TITLE_FONT_SIZE
          ),
        ];

    fadeInTargets(this.scene, [banner, ...texts], FADE_MS, () => {
      this.scene.time.delayedCall(VICTORY_BUTTON_DELAY_MS, () => {
        this.renderButtons(width, bannerY, callbacks);
      });
    });
  }

  private addBannerText(
    x: number,
    y: number,
    label: string,
    fontSize: number
  ): Phaser.GameObjects.Text {
    const text = this.scene.add.text(x, y, label, {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${fontSize}px`,
      color: colorToCss(TURN_BANNER_TEXT_COLOR),
    });
    text.setOrigin(0.5);
    text.setDepth(DEPTH_VICTORY_CUTSCENE);
    text.setScrollFactor(0);
    return text;
  }

  private renderButtons(width: number, bannerY: number, callbacks: VictoryCutsceneCallbacks): void {
    const x = width / 2 - LIFECYCLE_BUTTON_WIDTH / 2;
    const rematchY = bannerY + TURN_BANNER_HEIGHT + VICTORY_BUTTON_GAP;
    const returnY = verticalButtonY(rematchY, 1, LIFECYCLE_BUTTON_HEIGHT_SMALL, VICTORY_BUTTON_GAP);

    drawPillButton(
      this.scene,
      x,
      rematchY,
      LIFECYCLE_BUTTON_WIDTH,
      LIFECYCLE_BUTTON_HEIGHT_SMALL,
      'Rematch',
      BUTTON_STYLE,
      DEPTH_VICTORY_CUTSCENE,
      callbacks.onRematch,
      0
    );

    drawPillButton(
      this.scene,
      x,
      returnY,
      LIFECYCLE_BUTTON_WIDTH,
      LIFECYCLE_BUTTON_HEIGHT_SMALL,
      'Return to Match Settings',
      BUTTON_STYLE,
      DEPTH_VICTORY_CUTSCENE,
      callbacks.onReturnToSettings,
      0
    );
  }
}
