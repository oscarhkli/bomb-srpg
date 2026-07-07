import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { CONFIRM_DIALOG_DIM_ALPHA, CONFIRM_DIALOG_DIM_COLOR } from '../constants';
import ConfirmDialog from './ConfirmDialog';

function allGraphics(): ReturnType<typeof mockScene.add.graphics>[] {
  return mockScene.add.graphics.mock.results.map(
    r => r.value as ReturnType<typeof mockScene.add.graphics>
  );
}

function clickPointerdown(g: ReturnType<typeof mockScene.add.graphics>): void {
  const onPointerDown = g.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void;
  onPointerDown();
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('ConfirmDialog', () => {
  it('renders a dimmed rect and a "Confirm?" prompt when shown', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn());

    const [bg] = allGraphics();
    expect(bg!.fillStyle).toHaveBeenCalledWith(CONFIRM_DIALOG_DIM_COLOR, CONFIRM_DIALOG_DIM_ALPHA);
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Confirm?',
      expect.objectContaining({})
    );
  });

  it('pins every dialog element to the camera viewport (scrollFactor 0) so it stays screen-centered regardless of camera scroll', () => {
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(vi.fn(), vi.fn());

    allGraphics().forEach(g => expect(g.setScrollFactor).toHaveBeenCalledWith(0));
    const promptText = mockScene.add.text.mock.results[0]!.value as ReturnType<
      typeof mockScene.add.text
    >;
    expect(promptText.setScrollFactor).toHaveBeenCalledWith(0);
  });

  it('invokes onYes and hides when the Yes button is clicked', () => {
    const onYes = vi.fn();
    const onNo = vi.fn();
    const dialog = new ConfirmDialog(mockScene as never);

    dialog.show(onYes, onNo);

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

    dialog.show(onYes, onNo);

    const [, , noButtonGraphics] = allGraphics();
    clickPointerdown(noButtonGraphics!);

    expect(onNo).toHaveBeenCalledOnce();
    expect(onYes).not.toHaveBeenCalled();
    expect(dialog.isOpen).toBe(false);
  });

  it('hide() destroys all dialog objects', () => {
    const dialog = new ConfirmDialog(mockScene as never);
    dialog.show(vi.fn(), vi.fn());

    dialog.hide();

    allGraphics().forEach(g => expect(g.destroy).toHaveBeenCalled());
    expect(dialog.isOpen).toBe(false);
  });
});
