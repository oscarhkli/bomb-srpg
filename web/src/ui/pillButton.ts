import Phaser from 'phaser';
import { GAME_FONT_FAMILY } from '../constants';
import { colorToCss } from './gameObjectUtils';

export interface PillButtonStyle {
  fillColor: number;
  fillAlpha: number;
  borderColor: number;
  borderWidth: number;
}

// Draws a pill-shape button (rounded rect + border + centered label) per the shared styling
// used by TurnCommandPanel's Move/Bomb/Back buttons and ConfirmDialog's Yes/No buttons.
// Returns the created GameObjects so the caller can track them for later destroy().
export function drawPillButton(
  scene: Phaser.Scene,
  x: number,
  y: number,
  width: number,
  height: number,
  label: string,
  style: PillButtonStyle,
  depth: number,
  onClick?: () => void,
  scrollFactor?: number
): Phaser.GameObjects.GameObject[] {
  const g = scene.add.graphics();
  g.setDepth(depth);
  g.fillStyle(style.fillColor, style.fillAlpha);
  g.fillRoundedRect(x, y, width, height, height / 2);
  g.lineStyle(style.borderWidth, style.borderColor, 1);
  g.strokeRoundedRect(x, y, width, height, height / 2);

  const text = scene.add.text(x + width / 2, y + height / 2, label, {
    fontFamily: GAME_FONT_FAMILY,
    color: colorToCss(style.borderColor),
  });
  text.setOrigin(0.5);
  text.setDepth(depth);

  if (scrollFactor !== undefined) {
    g.setScrollFactor(scrollFactor);
    text.setScrollFactor(scrollFactor);
  }

  if (onClick) {
    const hitArea = new Phaser.Geom.Rectangle(x, y, width, height);
    g.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, px: number, py: number) =>
      Phaser.Geom.Rectangle.Contains(shape, px, py)
    );
    g.on('pointerdown', onClick);
  }

  return [g, text];
}
