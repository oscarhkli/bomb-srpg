import type Phaser from 'phaser';
import {
  CONFIRM_DIALOG_DIM_ALPHA,
  CONFIRM_DIALOG_DIM_COLOR,
  CONFIRM_DIALOG_HEIGHT,
  CONFIRM_DIALOG_WIDTH,
  DEPTH_CONFIRM_DIALOG,
  GAME_FONT_FAMILY,
  BUTTON_LABEL_YES,
  BUTTON_LABEL_NO,
  SPRITE_BUTTON_ROW_SPACING,
} from '../constants';
import SpriteButton, { buttonFrameSize } from './spriteButton';
import { verticalButtonY } from './pillButton';

const DIALOG_MARGIN = 12;

export default class ConfirmDialog {
  private objects: { destroy(): void }[] = [];

  constructor(private readonly scene: Phaser.Scene) {}

  get isOpen(): boolean {
    return this.objects.length > 0;
  }

  show(onYes: () => void, onNo: () => void, text: string): void {
    this.hide();

    const { width, height } = this.scene.cameras.main;
    const x = width / 2 - CONFIRM_DIALOG_WIDTH / 2;
    const y = height / 2 - CONFIRM_DIALOG_HEIGHT / 2;

    const bg = this.scene.add.graphics();
    bg.setDepth(DEPTH_CONFIRM_DIALOG);
    bg.setScrollFactor(0);
    bg.fillStyle(CONFIRM_DIALOG_DIM_COLOR, CONFIRM_DIALOG_DIM_ALPHA);
    bg.fillRect(x, y, CONFIRM_DIALOG_WIDTH, CONFIRM_DIALOG_HEIGHT);
    this.objects.push(bg);

    const promptText = this.scene.add.text(width / 2, y + DIALOG_MARGIN, text, {
      fontFamily: GAME_FONT_FAMILY,
      color: '#ffffff',
    });
    promptText.setOrigin(0.5);
    promptText.setDepth(DEPTH_CONFIRM_DIALOG);
    promptText.setScrollFactor(0);
    this.objects.push(promptText);

    const centerX = width / 2;
    const { height: buttonHeight } = buttonFrameSize(this.scene);
    const stackHeight = 2 * buttonHeight + SPRITE_BUTTON_ROW_SPACING;
    const firstCenterY = y + CONFIRM_DIALOG_HEIGHT - DIALOG_MARGIN - stackHeight + buttonHeight / 2;

    this.objects.push(
      this.createButton(centerX, firstCenterY, BUTTON_LABEL_YES, () => {
        this.hide();
        onYes();
      })
    );

    this.objects.push(
      this.createButton(
        centerX,
        verticalButtonY(firstCenterY, 1, buttonHeight, SPRITE_BUTTON_ROW_SPACING),
        BUTTON_LABEL_NO,
        () => {
          this.hide();
          onNo();
        }
      )
    );
  }

  private createButton(x: number, y: number, labelKey: string, onClick: () => void): SpriteButton {
    const button = new SpriteButton(this.scene, {
      x,
      y,
      labelKey,
      depth: DEPTH_CONFIRM_DIALOG,
      enabled: true,
      onClick,
    });
    button.container.setScrollFactor(0);
    return button;
  }

  hide(): void {
    this.objects.forEach(obj => obj.destroy());
    this.objects = [];
  }
}
