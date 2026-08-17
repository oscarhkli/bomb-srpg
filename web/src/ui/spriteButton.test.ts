import { describe, it, expect, beforeEach, vi } from 'vitest';
import type Phaser from 'phaser';
import { mockScene } from '../test/setup';
import { allContainers, allImages, clickPointerdown, firePointerEvent } from '../test/sceneHelpers';
import {
  BUTTON_TEXTURE_NEUTRAL,
  BUTTON_TEXTURE_SELECTED,
  BUTTON_TEXTURE_CLICKED,
  BUTTON_LABEL_MOVE,
  DEPTH_TURN_COMMAND_PANEL,
  SPRITE_BUTTON_CLICKED_LABEL_OFFSET_Y,
} from '../constants';
import SpriteButton from './spriteButton';

beforeEach(() => {
  vi.clearAllMocks();
});

function makeButton(overrides: Partial<{ enabled: boolean; onClick: () => void }> = {}) {
  const onClick = overrides.onClick ?? vi.fn();
  const button = new SpriteButton(mockScene as never, {
    x: 10,
    y: 20,
    labelKey: BUTTON_LABEL_MOVE,
    depth: DEPTH_TURN_COMMAND_PANEL,
    enabled: overrides.enabled ?? true,
    onClick,
  });
  return { button, onClick };
}

describe('SpriteButton', () => {
  it('renders a Container at (x, y) holding the Neutral button image and the label image', () => {
    makeButton();

    const [container] = allContainers();
    expect(mockScene.add.container).toHaveBeenCalledWith(10, 20);
    expect(container!.setDepth).toHaveBeenCalledWith(DEPTH_TURN_COMMAND_PANEL);

    const [buttonImage, labelImage] = allImages();
    expect(mockScene.add.image).toHaveBeenCalledWith(
      0,
      0,
      BUTTON_TEXTURE_NEUTRAL,
      `${BUTTON_TEXTURE_NEUTRAL}-frame`
    );
    expect(mockScene.add.image).toHaveBeenCalledWith(
      0,
      0,
      BUTTON_LABEL_MOVE,
      `${BUTTON_LABEL_MOVE}-frame`
    );
    expect(buttonImage!.setOrigin).toHaveBeenCalledWith(0.5, 0.5);
    expect(labelImage!.setOrigin).toHaveBeenCalledWith(0.5, 0.5);
    expect(container!.add).toHaveBeenCalledWith([buttonImage, labelImage]);
  });

  it('shifts the label depth above the button so the label renders on top', () => {
    makeButton();

    const [buttonImage, labelImage] = allImages();
    expect(labelImage!.setDepth).toHaveBeenCalled();
    const buttonDepth = buttonImage!.setDepth.mock.calls[0]![0] as number;
    const labelDepth = labelImage!.setDepth.mock.calls[0]![0] as number;
    expect(labelDepth).toBeGreaterThan(buttonDepth);
  });

  it('sets a top-left-based hit area (0, 0, width, height), not centered on the container origin', () => {
    makeButton();

    const [container] = allContainers();
    const [hitArea] = container!.setInteractive.mock.calls[0] as [Phaser.Geom.Rectangle];
    expect(hitArea).toMatchObject({ x: 0, y: 0, width: 100, height: 100 });
  });

  it('swaps to Selected on pointerover and back to Neutral on pointerout when enabled', () => {
    makeButton({ enabled: true });

    const [container] = allContainers();
    const [buttonImage] = allImages();
    firePointerEvent(container!, 'pointerover');
    expect(buttonImage!.setTexture).toHaveBeenCalledWith(
      BUTTON_TEXTURE_SELECTED,
      `${BUTTON_TEXTURE_SELECTED}-frame`
    );

    firePointerEvent(container!, 'pointerout');
    expect(buttonImage!.setTexture).toHaveBeenCalledWith(
      BUTTON_TEXTURE_NEUTRAL,
      `${BUTTON_TEXTURE_NEUTRAL}-frame`
    );
  });

  it('swaps to Clicked and shifts the label 2px down on pointerdown, invoking onClick', () => {
    const { onClick } = makeButton({ enabled: true });

    const [container] = allContainers();
    const [buttonImage, labelImage] = allImages();
    clickPointerdown(container!);

    expect(buttonImage!.setTexture).toHaveBeenCalledWith(
      BUTTON_TEXTURE_CLICKED,
      `${BUTTON_TEXTURE_CLICKED}-frame`
    );
    expect(labelImage!.y).toBe(SPRITE_BUTTON_CLICKED_LABEL_OFFSET_Y);
    expect(onClick).toHaveBeenCalledOnce();
  });

  it('reverts Clicked back to Neutral and the label to y=0 on pointerup outside', () => {
    makeButton({ enabled: true });

    const [container] = allContainers();
    const [buttonImage, labelImage] = allImages();
    clickPointerdown(container!);
    firePointerEvent(container!, 'pointerupoutside');

    expect(buttonImage!.setTexture).toHaveBeenLastCalledWith(
      BUTTON_TEXTURE_NEUTRAL,
      `${BUTTON_TEXTURE_NEUTRAL}-frame`
    );
    expect(labelImage!.y).toBe(0);
  });

  it('does not wire pointer handlers or fire onClick when created disabled', () => {
    const { onClick } = makeButton({ enabled: false });

    const [container] = allContainers();
    expect(container!.on).not.toHaveBeenCalled();
    expect(container!.disableInteractive).toHaveBeenCalled();
    expect(onClick).not.toHaveBeenCalled();
  });

  it('desaturates the container while disabled and clears the filter once re-enabled', () => {
    const { button } = makeButton({ enabled: false });

    const [container] = allContainers();
    expect(container!.enableFilters).toHaveBeenCalled();
    expect(container!.filters.internal.addColorMatrix).toHaveBeenCalled();

    button.setEnabled(true);
    expect(container!.filters.internal.clear).toHaveBeenCalled();
    expect(container!.setInteractive).toHaveBeenCalled();
  });

  it('destroy() destroys the container', () => {
    const { button } = makeButton();

    button.destroy();

    const [container] = allContainers();
    expect(container!.destroy).toHaveBeenCalled();
  });
});
