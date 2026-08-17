import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { allContainers, allGraphics, clickPointerdown, firstText } from '../test/sceneHelpers';
import {
  CONFIRM_DIALOG_DIM_ALPHA,
  CONFIRM_DIALOG_DIM_COLOR,
  CONFIRM_DIALOG_WIDTH,
  CONFIRM_DIALOG_HEIGHT,
  BUTTON_LABEL_YES,
  BUTTON_LABEL_NO,
} from '../constants';
import ConfirmDialog from './ConfirmDialog';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('ConfirmDialog', () => {
  it('centers on the 640x360 camera', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn(), 'Confirm?');

    const [bg] = allGraphics();
    expect(bg!.fillRect).toHaveBeenCalledWith(
      640 / 2 - CONFIRM_DIALOG_WIDTH / 2,
      360 / 2 - CONFIRM_DIALOG_HEIGHT / 2,
      CONFIRM_DIALOG_WIDTH,
      CONFIRM_DIALOG_HEIGHT
    );
  });

  it('renders a dimmed rect and a "Confirm?" prompt when shown', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn(), 'Confirm?');

    const [bg] = allGraphics();
    expect(bg!.fillStyle).toHaveBeenCalledWith(CONFIRM_DIALOG_DIM_COLOR, CONFIRM_DIALOG_DIM_ALPHA);
    expect(bg!.fillRect).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      CONFIRM_DIALOG_WIDTH,
      CONFIRM_DIALOG_HEIGHT
    );
    expect(CONFIRM_DIALOG_WIDTH).toBe(240);
    expect(CONFIRM_DIALOG_HEIGHT).toBe(144);
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Confirm?',
      expect.objectContaining({})
    );
  });

  it('renders a caller-supplied prompt text instead of the default', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn(), 'Confirm to end this turn?');

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Confirm to end this turn?',
      expect.objectContaining({})
    );
  });

  it('stacks Yes above No, both SpriteButtons, centered on the dialog', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn(), 'Confirm?');

    expect(allContainers()).toHaveLength(2);
    expect(mockScene.add.image).toHaveBeenCalledWith(
      0,
      0,
      BUTTON_LABEL_YES,
      `${BUTTON_LABEL_YES}-frame`
    );
    expect(mockScene.add.image).toHaveBeenCalledWith(
      0,
      0,
      BUTTON_LABEL_NO,
      `${BUTTON_LABEL_NO}-frame`
    );
    const [yesContainer, noContainer] = allContainers();
    const centerX = mockScene.add.container.mock.calls[0]![0]!;
    expect(mockScene.add.container).toHaveBeenNthCalledWith(1, centerX, expect.any(Number));
    expect(mockScene.add.container).toHaveBeenNthCalledWith(2, centerX, expect.any(Number));
    expect(noContainer!.y).toBeGreaterThan(yesContainer!.y);
  });

  it('pins every dialog element to the camera viewport (scrollFactor 0) so it stays screen-centered regardless of camera scroll', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn(), 'Confirm?');

    allGraphics().forEach(g => expect(g.setScrollFactor).toHaveBeenCalledWith(0));
    allContainers().forEach(c => expect(c.setScrollFactor).toHaveBeenCalledWith(0));
    expect(firstText().setScrollFactor).toHaveBeenCalledWith(0);
  });

  it('invokes onYes and hides when the Yes button is clicked', () => {
    const onYes = vi.fn();
    const onNo = vi.fn();
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(onYes, onNo, 'Confirm?');

    const [yesButtonContainer] = allContainers();
    clickPointerdown(yesButtonContainer!);

    expect(onYes).toHaveBeenCalledOnce();
    expect(onNo).not.toHaveBeenCalled();
    expect(dialog.isOpen).toBe(false);
  });

  it('invokes onNo and hides when the No button is clicked', () => {
    const onYes = vi.fn();
    const onNo = vi.fn();
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(onYes, onNo, 'Confirm?');

    const [, noButtonContainer] = allContainers();
    clickPointerdown(noButtonContainer!);

    expect(onNo).toHaveBeenCalledOnce();
    expect(onYes).not.toHaveBeenCalled();
    expect(dialog.isOpen).toBe(false);
  });

  it('hide() destroys all dialog objects', () => {
    const dialog = new ConfirmDialog(mockScene as never);
    dialog.show(vi.fn(), vi.fn(), 'Confirm?');

    dialog.hide();

    allGraphics().forEach(g => expect(g.destroy).toHaveBeenCalled());
    allContainers().forEach(c => expect(c.destroy).toHaveBeenCalled());
    expect(dialog.isOpen).toBe(false);
  });
});
