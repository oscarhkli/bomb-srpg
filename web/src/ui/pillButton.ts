import Phaser from 'phaser';
import { GAME_FONT_FAMILY } from '../constants';
import { colorToCss } from './gameObjectUtils';

export interface PillButtonStyle {
  fillColor: number;
  fillAlpha: number;
  borderColor: number;
  borderWidth: number;
}

// Shared by any clickable Graphics rect (pill buttons here, and MatchSummaryPanel's square
// "≡" icon button) so the hit-area/interactive wiring only lives in one place.
export function attachRectClickHandler(
  target: Phaser.GameObjects.Graphics,
  x: number,
  y: number,
  width: number,
  height: number,
  onClick: () => void
): void {
  const hitArea = new Phaser.Geom.Rectangle(x, y, width, height);
  target.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, px: number, py: number) =>
    Phaser.Geom.Rectangle.Contains(shape, px, py)
  );
  target.on('pointerdown', onClick);
}

// Y position of the i-th button in a vertical stack starting at startY, each buttonHeight tall
// with spacing between them. Shared by MatchSummaryPanel (bottom-anchored block) and
// VictoryCutscene (top-anchored pair) so their spacing math can't silently drift apart.
export function verticalButtonY(
  startY: number,
  index: number,
  buttonHeight: number,
  spacing: number
): number {
  return startY + index * (buttonHeight + spacing);
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
): (Phaser.GameObjects.Graphics | Phaser.GameObjects.Text)[] {
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
    attachRectClickHandler(g, x, y, width, height, onClick);
  }

  return [g, text];
}
