import type Phaser from 'phaser';

export function destroyAll(objects: { destroy(): void }[]): void {
  objects.forEach(o => o.destroy());
  objects.length = 0;
}

export function colorToCss(color: number): string {
  return `#${color.toString(16).padStart(6, '0')}`;
}

export function createFilledRect(
  scene: Phaser.Scene,
  x: number,
  y: number,
  width: number,
  height: number,
  color: number,
  depth: number
): Phaser.GameObjects.Graphics {
  const rect = scene.add.graphics();
  rect.setDepth(depth);
  rect.setScrollFactor(0);
  rect.fillStyle(color);
  rect.fillRect(x, y, width, height);
  return rect;
}

export function fadeInTargets(
  scene: Phaser.Scene,
  targets: { alpha: number }[],
  duration: number,
  onComplete: () => void
): void {
  targets.forEach(target => {
    target.alpha = 0;
  });
  scene.tweens.add({ targets, alpha: 1, duration, onComplete });
}
