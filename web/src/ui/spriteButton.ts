import Phaser from 'phaser';
import {
  BUTTON_TEXTURE_NEUTRAL,
  BUTTON_TEXTURE_SELECTED,
  BUTTON_TEXTURE_CLICKED,
  SPRITE_BUTTON_CLICKED_LABEL_OFFSET_Y,
} from '../constants';
import { firstNonBaseFrame } from '../rendering/spriteFrames';

export interface SpriteButtonConfig {
  x: number;
  y: number;
  labelKey: string;
  depth: number;
  enabled: boolean;
  onClick: () => void;
}

// Neutral's trimmed frame size — shared by every SpriteButton's hit area and by callers laying
// out a stack of them (TurnCommandPanel, ConfirmDialog).
export function buttonFrameSize(scene: Phaser.Scene): { width: number; height: number } {
  const frame = firstNonBaseFrame(scene, BUTTON_TEXTURE_NEUTRAL);
  const { width, height } = scene.textures.getFrame(BUTTON_TEXTURE_NEUTRAL, frame);
  return { width, height };
}

// A composite Button+Label component swapping between Neutral/Selected/Clicked/Disabled sprite
// states. Both GameObjects sit at local (0, 0) with origin (0.5, 0.5) so trimmed atlas frames
// self-align regardless of each state's differing trim.
export default class SpriteButton {
  readonly container: Phaser.GameObjects.Container;
  private readonly buttonImage: Phaser.GameObjects.Image;
  private readonly labelImage: Phaser.GameObjects.Image;
  private readonly hitArea: Phaser.Geom.Rectangle;
  private readonly onClick: () => void;
  private hovered = false;

  constructor(
    private readonly scene: Phaser.Scene,
    config: SpriteButtonConfig
  ) {
    this.onClick = config.onClick;

    this.container = scene.add.container(config.x, config.y);
    this.container.setDepth(config.depth);
    // Enabled up front so `filters` is non-null regardless of which state setEnabled starts in.
    this.container.enableFilters();

    this.buttonImage = scene.add
      .image(0, 0, BUTTON_TEXTURE_NEUTRAL, firstNonBaseFrame(scene, BUTTON_TEXTURE_NEUTRAL))
      .setOrigin(0.5, 0.5)
      .setDepth(0);
    this.labelImage = scene.add
      .image(0, 0, config.labelKey, firstNonBaseFrame(scene, config.labelKey))
      .setOrigin(0.5, 0.5)
      .setDepth(1);
    this.container.add([this.buttonImage, this.labelImage]);

    const { width, height } = buttonFrameSize(scene);
    this.container.setSize(width, height);
    // Container.displayOriginX/Y is locked to width/2, height/2 and gets added to the pointer's
    // local coordinate before hit-testing, so this rectangle must be top-left-based, not centered.
    this.hitArea = new Phaser.Geom.Rectangle(0, 0, width, height);

    this.setEnabled(config.enabled);
  }

  setEnabled(enabled: boolean): void {
    this.container.off('pointerover');
    this.container.off('pointerout');
    this.container.off('pointerdown');
    this.container.off('pointerup');
    this.container.off('pointerupoutside');

    this.container.filters!.internal.clear();

    if (!enabled) {
      this.hovered = false;
      this.setButtonTexture(BUTTON_TEXTURE_NEUTRAL);
      this.labelImage.y = 0;
      this.container.disableInteractive();
      this.container.filters!.internal.addColorMatrix().colorMatrix.desaturate();
      return;
    }

    this.container.setInteractive(
      this.hitArea,
      (shape: Phaser.Geom.Rectangle, px: number, py: number) =>
        Phaser.Geom.Rectangle.Contains(shape, px, py)
    );
    this.container.on('pointerover', () => this.onPointerOver());
    this.container.on('pointerout', () => this.onPointerOut());
    this.container.on('pointerdown', () => this.onPointerDown());
    this.container.on('pointerup', () => this.onPointerUp());
    this.container.on('pointerupoutside', () => this.onPointerUp());
  }

  destroy(): void {
    this.container.destroy();
  }

  private onPointerOver(): void {
    this.hovered = true;
    this.setButtonTexture(BUTTON_TEXTURE_SELECTED);
  }

  private onPointerOut(): void {
    this.hovered = false;
    this.setButtonTexture(BUTTON_TEXTURE_NEUTRAL);
    this.labelImage.y = 0;
  }

  private onPointerDown(): void {
    this.setButtonTexture(BUTTON_TEXTURE_CLICKED);
    this.labelImage.y = SPRITE_BUTTON_CLICKED_LABEL_OFFSET_Y;
    this.onClick();
  }

  private onPointerUp(): void {
    this.labelImage.y = 0;
    this.setButtonTexture(this.hovered ? BUTTON_TEXTURE_SELECTED : BUTTON_TEXTURE_NEUTRAL);
  }

  private setButtonTexture(key: string): void {
    this.buttonImage.setTexture(key, firstNonBaseFrame(this.scene, key));
  }
}
