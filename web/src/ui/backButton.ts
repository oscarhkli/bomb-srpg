import type Phaser from 'phaser';
import {
  BACK_BUTTON_SIZE,
  BACK_BUTTON_COLOR,
  BACK_BUTTON_GLYPH,
  BACK_BUTTON_GLYPH_FONT_SIZE,
  SETTINGS_CORNER_RADIUS,
  GAME_FONT_FAMILY,
} from '../constants';
import { colorToCss } from './gameObjectUtils';
import { attachRectClickHandler } from './pillButton';

// Shared back-navigation control: a rounded-square button with a centered ⮐ glyph.
export function drawBackButton(
  scene: Phaser.Scene,
  x: number,
  y: number,
  onClick: () => void
): (Phaser.GameObjects.Graphics | Phaser.GameObjects.Text)[] {
  const g = scene.add.graphics();
  g.fillStyle(BACK_BUTTON_COLOR);
  g.fillRoundedRect(x, y, BACK_BUTTON_SIZE, BACK_BUTTON_SIZE, SETTINGS_CORNER_RADIUS);

  const text = scene.add.text(
    x + BACK_BUTTON_SIZE / 2,
    y + BACK_BUTTON_SIZE / 2,
    BACK_BUTTON_GLYPH,
    {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${BACK_BUTTON_GLYPH_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
    }
  );
  text.setOrigin(0.5);

  attachRectClickHandler(g, x, y, BACK_BUTTON_SIZE, BACK_BUTTON_SIZE, onClick);

  return [g, text];
}
