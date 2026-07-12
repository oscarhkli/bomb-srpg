import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { allGraphics, clickPointerdown, firstText } from '../test/sceneHelpers';
import {
  CONFIRM_DIALOG_DIM_ALPHA,
  CONFIRM_DIALOG_DIM_COLOR,
  CONFIRM_DIALOG_WIDTH,
  CONFIRM_DIALOG_HEIGHT,
} from '../constants';
import ConfirmDialog from './ConfirmDialog';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('ConfirmDialog', () => {
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

  it('pins every dialog element to the camera viewport (scrollFactor 0) so it stays screen-centered regardless of camera scroll', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn(), 'Confirm?');

    allGraphics().forEach(g => expect(g.setScrollFactor).toHaveBeenCalledWith(0));
    expect(firstText().setScrollFactor).toHaveBeenCalledWith(0);
  });

  it('invokes onYes and hides when the Yes button is clicked', () => {
    const onYes = vi.fn();
    const onNo = vi.fn();
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(onYes, onNo, 'Confirm?');

    // graphics results: [0]=dim bg, [1]=Yes button graphics, [2]=No button graphics
    const [, yesButtonGraphics] = allGraphics();
    clickPointerdown(yesButtonGraphics!);

    expect(onYes).toHaveBeenCalledOnce();
    expect(onNo).not.toHaveBeenCalled();
    expect(dialog.isOpen).toBe(false);
  });

  it('invokes onNo and hides when the No button is clicked', () => {
    const onYes = vi.fn();
    const onNo = vi.fn();
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(onYes, onNo, 'Confirm?');

    const [, , noButtonGraphics] = allGraphics();
    clickPointerdown(noButtonGraphics!);

    expect(onNo).toHaveBeenCalledOnce();
    expect(onYes).not.toHaveBeenCalled();
    expect(dialog.isOpen).toBe(false);
  });

  it('hide() destroys all dialog objects', () => {
    const dialog = new ConfirmDialog(mockScene as never);
    dialog.show(vi.fn(), vi.fn(), 'Confirm?');

    dialog.hide();

    allGraphics().forEach(g => expect(g.destroy).toHaveBeenCalled());
    expect(dialog.isOpen).toBe(false);
  });
});
