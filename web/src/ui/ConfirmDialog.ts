import type Phaser from 'phaser';
import {
  CONFIRM_DIALOG_DIM_ALPHA,
  CONFIRM_DIALOG_DIM_COLOR,
  CONFIRM_DIALOG_HEIGHT,
  CONFIRM_DIALOG_WIDTH,
  DEPTH_CONFIRM_DIALOG,
  GAME_FONT_FAMILY,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_HEIGHT,
  PANEL_BUTTON_WIDTH,
} from '../constants';
import { drawPillButton } from './pillButton';

const DIALOG_MARGIN = 12;

export default class ConfirmDialog {
  private objects: Phaser.GameObjects.GameObject[] = [];

  constructor(private readonly scene: Phaser.Scene) {}

  get isOpen(): boolean {
    return this.objects.length > 0;
  }

  show(onYes: () => void, onNo: () => void): void {
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

    const promptText = this.scene.add.text(width / 2, y + DIALOG_MARGIN, 'Confirm?', {
      fontFamily: GAME_FONT_FAMILY,
      color: '#ffffff',
    });
    promptText.setOrigin(0.5);
    promptText.setDepth(DEPTH_CONFIRM_DIALOG);
    promptText.setScrollFactor(0);
    this.objects.push(promptText);

    const buttonY = y + CONFIRM_DIALOG_HEIGHT - PANEL_BUTTON_HEIGHT - DIALOG_MARGIN;
    const buttonStyle = {
      fillColor: PANEL_BUTTON_FILL_COLOR,
      fillAlpha: PANEL_BUTTON_FILL_ALPHA,
      borderColor: PANEL_BUTTON_BORDER_COLOR,
      borderWidth: PANEL_BUTTON_BORDER_WIDTH,
    };

    this.objects.push(
      ...drawPillButton(
        this.scene,
        x + DIALOG_MARGIN,
        buttonY,
        PANEL_BUTTON_WIDTH,
        PANEL_BUTTON_HEIGHT,
        'Yes',
        buttonStyle,
        DEPTH_CONFIRM_DIALOG,
        () => {
          this.hide();
          onYes();
        },
        0
      )
    );

    this.objects.push(
      ...drawPillButton(
        this.scene,
        x + CONFIRM_DIALOG_WIDTH - PANEL_BUTTON_WIDTH - DIALOG_MARGIN,
        buttonY,
        PANEL_BUTTON_WIDTH,
        PANEL_BUTTON_HEIGHT,
        'No',
        buttonStyle,
        DEPTH_CONFIRM_DIALOG,
        () => {
          this.hide();
          onNo();
        },
        0
      )
    );
  }

  hide(): void {
    this.objects.forEach(obj => obj.destroy());
    this.objects = [];
  }
}
