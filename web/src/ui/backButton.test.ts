import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { firstGraphics, firstText, pointerDownOf } from '../test/sceneHelpers';
import {
  BACK_BUTTON_SIZE,
  BACK_BUTTON_COLOR,
  BACK_BUTTON_GLYPH,
  SETTINGS_CORNER_RADIUS,
} from '../constants';
import { drawBackButton } from './backButton';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('drawBackButton', () => {
  it('draws a BACK_BUTTON_COLOR rounded square at the given position', () => {
    drawBackButton(mockScene as never, 48, 48, vi.fn());

    const g = firstGraphics();
    expect(g.fillStyle).toHaveBeenCalledWith(BACK_BUTTON_COLOR);
    expect(g.fillRoundedRect).toHaveBeenCalledWith(
      48,
      48,
      BACK_BUTTON_SIZE,
      BACK_BUTTON_SIZE,
      SETTINGS_CORNER_RADIUS
    );
  });

  it('centers the glyph text on the button', () => {
    drawBackButton(mockScene as never, 48, 48, vi.fn());

    expect(mockScene.add.text).toHaveBeenCalledWith(
      48 + BACK_BUTTON_SIZE / 2,
      48 + BACK_BUTTON_SIZE / 2,
      BACK_BUTTON_GLYPH,
      expect.objectContaining({})
    );
    expect(firstText().setOrigin).toHaveBeenCalledWith(0.5);
  });

  it('invokes onClick when the button area is clicked', () => {
    const onClick = vi.fn();
    drawBackButton(mockScene as never, 48, 48, onClick);

    pointerDownOf(firstGraphics())();

    expect(onClick).toHaveBeenCalled();
  });

  it('returns both created GameObjects for the caller to track/destroy', () => {
    const objects = drawBackButton(mockScene as never, 48, 48, vi.fn());

    expect(objects).toEqual([firstGraphics(), firstText()]);
  });
});
