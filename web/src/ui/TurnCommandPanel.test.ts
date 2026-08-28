import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import { allContainers, allGraphics, clickPointerdown, lastContainers } from '../test/sceneHelpers';
import {
  GAME_CONTROL_REGION_X,
  GAME_CONTROL_REGION_WIDTH,
  GAME_CONTROL_REGION_HEIGHT,
  TURN_COMMAND_PANEL_BOTTOM_MARGIN,
  SPRITE_BUTTON_ROW_SPACING,
  BUTTON_LABEL_MOVE,
  BUTTON_LABEL_BOMB,
  BUTTON_LABEL_BACK,
  ALLOWED_TILE_MOVE_COLOR,
  ALLOWED_TILE_MOVE_ALPHA,
  ALLOWED_TILE_BOMB_COLOR,
} from '../constants';
import TurnCommandPanel from './TurnCommandPanel';
import type { Coordinate, TurnCmdType } from '../types/api';
import { makeUnit } from '../test/fixtures';

function makePanel(overrides: Partial<Record<string, unknown>> = {}) {
  const defaultGetAllowedTiles = vi.fn<
    (unitId: number, turnCmdType: TurnCmdType) => Promise<Coordinate[]>
  >(() => Promise.resolve([]));
  const defaultOnError = vi.fn();
  const defaultOnConfirmedSubmit = vi.fn();
  const defaultShowConfirm = vi.fn<(onYes: () => void, onNo: () => void) => void>();
  const defaultHideConfirm = vi.fn();
  const defaultIsConfirmOpen = vi.fn(() => false);
  const defaultIsLocked = vi.fn(() => false);
  const callbacks = {
    getAllowedTiles: defaultGetAllowedTiles,
    onError: defaultOnError,
    onConfirmedSubmit: defaultOnConfirmedSubmit,
    showConfirm: defaultShowConfirm,
    hideConfirm: defaultHideConfirm,
    isConfirmOpen: defaultIsConfirmOpen,
    isLocked: defaultIsLocked,
    ...overrides,
  };
  const panel = new TurnCommandPanel(mockScene as never, callbacks);
  return {
    panel,
    getAllowedTiles: callbacks.getAllowedTiles,
    onError: callbacks.onError,
    onConfirmedSubmit: callbacks.onConfirmedSubmit,
    showConfirm: callbacks.showConfirm,
    hideConfirm: callbacks.hideConfirm,
    isConfirmOpen: callbacks.isConfirmOpen,
    isLocked: callbacks.isLocked,
  };
}

function latestConfirmCallbacks(showConfirm: ReturnType<typeof vi.fn>): {
  onYes: () => void;
  onNo: () => void;
} {
  const lastCall = showConfirm.mock.calls.at(-1) as [() => void, () => void];
  return { onYes: lastCall[0], onNo: lastCall[1] };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('TurnCommandPanel', () => {
  it('stacks Move/Bomb/Back centered in GameControlRegion, bottom-anchored with a 40px margin', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit());

    // GameControlRegion is 480..640 wide (centerX=560); the mocked button_neutral frame is
    // 100x100, so the 3-row stack (with 4px gaps) is 308 tall, bottom-anchored 40px above y=360.
    const centerX = GAME_CONTROL_REGION_X + GAME_CONTROL_REGION_WIDTH / 2;
    const buttonHeight = 100;
    const stackHeight = 3 * buttonHeight + 2 * SPRITE_BUTTON_ROW_SPACING;
    const panelBottomY = GAME_CONTROL_REGION_HEIGHT - TURN_COMMAND_PANEL_BOTTOM_MARGIN;
    const firstCenterY = panelBottomY - stackHeight + buttonHeight / 2;

    expect(mockScene.add.container).toHaveBeenNthCalledWith(1, centerX, firstCenterY);
    expect(mockScene.add.container).toHaveBeenNthCalledWith(
      2,
      centerX,
      firstCenterY + buttonHeight + SPRITE_BUTTON_ROW_SPACING
    );
    expect(mockScene.add.container).toHaveBeenNthCalledWith(
      3,
      centerX,
      firstCenterY + 2 * (buttonHeight + SPRITE_BUTTON_ROW_SPACING)
    );
  });

  it('draws three SpriteButtons with the Move/Bomb/Back labels when opened for a fresh unit', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit());

    expect(allContainers()).toHaveLength(3);
    expect(mockScene.add.image).toHaveBeenCalledWith(
      0,
      0,
      BUTTON_LABEL_MOVE,
      `${BUTTON_LABEL_MOVE}-frame`
    );
    expect(mockScene.add.image).toHaveBeenCalledWith(
      0,
      0,
      BUTTON_LABEL_BOMB,
      `${BUTTON_LABEL_BOMB}-frame`
    );
    expect(mockScene.add.image).toHaveBeenCalledWith(
      0,
      0,
      BUTTON_LABEL_BACK,
      `${BUTTON_LABEL_BACK}-frame`
    );
  });

  it('renders moveButton disabled (desaturated, non-interactive) when unit.hasMoved is true', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit({ hasMoved: true }));

    const [moveButtonContainer] = allContainers();
    expect(moveButtonContainer!.disableInteractive).toHaveBeenCalled();
    expect(moveButtonContainer!.filters.internal.addColorMatrix).toHaveBeenCalled();
    expect(moveButtonContainer!.on).not.toHaveBeenCalledWith('pointerdown', expect.any(Function));
  });

  it('renders placeBombButton disabled when unit.hasUsedSkill is true', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit({ hasUsedSkill: true }));

    const [, placeBombButtonContainer] = allContainers();
    expect(placeBombButtonContainer!.disableInteractive).toHaveBeenCalled();
  });

  it('renders placeBombButton disabled when unit has no bomb charge left, even with hasUsedSkill false', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit({ hasUsedSkill: false, maxBombCount: 2, bombUsed: 2 }));

    const [, placeBombButtonContainer] = allContainers();
    expect(placeBombButtonContainer!.disableInteractive).toHaveBeenCalled();
  });

  it('keeps placeBombButton disabled across a turn boundary while the placed bomb has not detonated, then re-enables once it has', () => {
    const { panel } = makePanel();

    // Next turn opens: hasUsedSkill resets, but bombUsed (server truth) still reflects
    // the outstanding bomb — the button must stay disabled.
    panel.openFor(makeUnit({ hasUsedSkill: false, maxBombCount: 1, bombUsed: 1 }));
    const [, stillDisabled] = lastContainers(3);
    expect(stillDisabled!.disableInteractive).toHaveBeenCalled();

    // The bomb has since detonated: bombUsed resets — button re-enables.
    panel.openFor(makeUnit({ hasUsedSkill: false, maxBombCount: 1, bombUsed: 0 }));
    const [, reenabled] = lastContainers(3);
    expect(reenabled!.disableInteractive).not.toHaveBeenCalled();
  });

  it('renders placeBombButton enabled when unit has an unused action and a bomb charge left', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit({ hasUsedSkill: false, maxBombCount: 2, bombUsed: 1 }));

    const [, placeBombButtonContainer] = allContainers();
    expect(placeBombButtonContainer!.disableInteractive).not.toHaveBeenCalled();
  });

  it('hides the panel when backButton is clicked with nothing to roll back', () => {
    const { panel } = makePanel();
    panel.openFor(makeUnit());

    const [, , backButtonContainer] = allContainers();
    clickPointerdown(backButtonContainer!);

    allContainers().forEach(c => expect(c.destroy).toHaveBeenCalled());
  });

  it('closeImmediately destroys every panel object', () => {
    const { panel } = makePanel();
    panel.openFor(makeUnit());

    panel.closeImmediately();

    allContainers().forEach(c => expect(c.destroy).toHaveBeenCalled());
  });

  it('re-opening for a different unit closes the previous panel first', () => {
    const { panel } = makePanel();
    panel.openFor(makeUnit({ id: 1 }));
    const firstButtons = allContainers();

    panel.openFor(makeUnit({ id: 2 }));

    firstButtons.forEach(c => expect(c.destroy).toHaveBeenCalled());
  });

  it('fetches and renders allowedTiles with move styling when moveButton is clicked', async () => {
    const tiles: Coordinate[] = [
      { x: 1, y: 0 },
      { x: 2, y: 0 },
    ];
    const { panel, getAllowedTiles } = makePanel({
      getAllowedTiles: vi.fn().mockResolvedValue(tiles),
    });
    const unit = makeUnit({ id: 5 });
    panel.openFor(unit);

    const [moveButtonContainer] = allContainers();
    clickPointerdown(moveButtonContainer!);
    await Promise.resolve();
    await Promise.resolve();

    expect(getAllowedTiles).toHaveBeenCalledWith(5, 'move');
    const overlayGraphics = allGraphics();
    expect(overlayGraphics).toHaveLength(2);
    overlayGraphics.forEach(g => {
      expect(g.fillStyle).toHaveBeenCalledWith(ALLOWED_TILE_MOVE_COLOR, ALLOWED_TILE_MOVE_ALPHA);
    });
  });

  it('backButton hides only the allowedTiles overlay, keeping the panel open, when popping past allowedTilesShown', async () => {
    const tiles: Coordinate[] = [{ x: 1, y: 0 }];
    const { panel } = makePanel({ getAllowedTiles: vi.fn().mockResolvedValue(tiles) });
    panel.openFor(makeUnit());

    const [moveButtonContainer, , backButtonContainer] = allContainers();
    clickPointerdown(moveButtonContainer!);
    await Promise.resolve();
    await Promise.resolve();

    const overlayGraphics = allGraphics();
    expect(overlayGraphics).toHaveLength(1);

    clickPointerdown(backButtonContainer!);

    overlayGraphics.forEach(g => expect(g.destroy).toHaveBeenCalled());
    // The panel's own 3 buttons are untouched — only the overlay was popped, not the whole panel.
    [moveButtonContainer, backButtonContainer].forEach(c =>
      expect(c!.destroy).not.toHaveBeenCalled()
    );
  });

  it('fetches and renders allowedTiles with placeBomb styling when placeBombButton is clicked', async () => {
    const tiles: Coordinate[] = [{ x: 1, y: 0 }];
    const { panel, getAllowedTiles } = makePanel({
      getAllowedTiles: vi.fn().mockResolvedValue(tiles),
    });
    const unit = makeUnit({ id: 5 });
    panel.openFor(unit);

    const [, placeBombButtonContainer] = allContainers();
    clickPointerdown(placeBombButtonContainer!);
    await Promise.resolve();
    await Promise.resolve();

    expect(getAllowedTiles).toHaveBeenCalledWith(5, 'placeBomb');
    const [overlayGraphics] = allGraphics();
    expect(overlayGraphics!.fillStyle).toHaveBeenCalledWith(
      ALLOWED_TILE_BOMB_COLOR,
      expect.any(Number)
    );
  });

  it('calls onError and shows no overlay when getAllowedTiles rejects', async () => {
    const { panel, onError } = makePanel({
      getAllowedTiles: vi.fn().mockRejectedValue(new Error('network down')),
    });
    panel.openFor(makeUnit());

    const [moveButtonContainer] = allContainers();
    clickPointerdown(moveButtonContainer!);
    await Promise.resolve();
    await Promise.resolve();

    expect(onError).toHaveBeenCalledWith('network down');
    expect(allGraphics()).toHaveLength(0); // no overlay tiles drawn
  });

  it('calls showConfirm with a target-bound onYes; invoking it calls onConfirmedSubmit', async () => {
    const target: Coordinate = { x: 1, y: 0 };
    const { panel, showConfirm, onConfirmedSubmit } = makePanel({
      getAllowedTiles: vi.fn().mockResolvedValue([target]),
    });
    const unit = makeUnit({ id: 9 });
    panel.openFor(unit);

    const [moveButtonContainer] = allContainers();
    clickPointerdown(moveButtonContainer!);
    await Promise.resolve();
    await Promise.resolve();

    const [overlayTileGraphics] = allGraphics();
    clickPointerdown(overlayTileGraphics!);

    expect(showConfirm).toHaveBeenCalledWith(expect.any(Function), expect.any(Function));
    const { onYes } = latestConfirmCallbacks(showConfirm);
    onYes();

    expect(onConfirmedSubmit).toHaveBeenCalledWith({ type: 'move', unitId: 9, target });
  });

  it('ignores a second tile click while a confirm dialog is already open', async () => {
    const tiles: Coordinate[] = [
      { x: 1, y: 0 },
      { x: 2, y: 0 },
    ];
    const isConfirmOpen = vi.fn(() => false);
    const { panel, showConfirm } = makePanel({
      getAllowedTiles: vi.fn().mockResolvedValue(tiles),
      isConfirmOpen,
    });
    panel.openFor(makeUnit({ id: 9 }));

    const [moveButtonContainer] = allContainers();
    clickPointerdown(moveButtonContainer!);
    await Promise.resolve();
    await Promise.resolve();

    const [firstTile, secondTile] = allGraphics();
    clickPointerdown(firstTile!);
    expect(showConfirm).toHaveBeenCalledTimes(1);

    isConfirmOpen.mockReturnValue(true);
    clickPointerdown(secondTile!);

    expect(showConfirm).toHaveBeenCalledTimes(1);
  });

  it('invoking the onNo passed to showConfirm re-fetches and re-shows the allowedTiles overlay', async () => {
    const target: Coordinate = { x: 1, y: 0 };
    const getAllowedTiles = vi.fn().mockResolvedValue([target]);
    const { panel, showConfirm } = makePanel({ getAllowedTiles });
    const unit = makeUnit({ id: 9 });
    panel.openFor(unit);

    const [moveButtonContainer] = allContainers();
    clickPointerdown(moveButtonContainer!);
    await Promise.resolve();
    await Promise.resolve();

    const [overlayTileGraphics] = allGraphics();
    clickPointerdown(overlayTileGraphics!);

    const { onNo } = latestConfirmCallbacks(showConfirm);
    onNo();
    await Promise.resolve();
    await Promise.resolve();

    // "No" discards the recolored tile and re-renders the overlay — the caller-side cache
    // (not this class) is what makes the re-fetch a no-op network-wise in the real app.
    expect(getAllowedTiles).toHaveBeenCalledTimes(2);
    expect(overlayTileGraphics!.destroy).toHaveBeenCalled();
    const newOverlayGraphics = allGraphics().at(-1);
    expect(newOverlayGraphics).toBeDefined();
    expect(newOverlayGraphics).not.toBe(overlayTileGraphics);
  });

  describe('isLocked guard (spec003-log issue #4 / spec008 interaction lock contract)', () => {
    it('ignores moveButton/placeBombButton clicks while locked', async () => {
      const isLocked = vi.fn(() => true);
      const { panel, getAllowedTiles } = makePanel({ isLocked });
      panel.openFor(makeUnit());

      const [moveButtonContainer, placeBombButtonContainer] = allContainers();
      clickPointerdown(moveButtonContainer!);
      clickPointerdown(placeBombButtonContainer!);
      await Promise.resolve();
      await Promise.resolve();

      expect(getAllowedTiles).not.toHaveBeenCalled();
    });

    it('ignores backButton clicks while locked', () => {
      const tiles: Coordinate[] = [{ x: 1, y: 0 }];
      const isLocked = vi.fn(() => false);
      const { panel } = makePanel({ isLocked, getAllowedTiles: vi.fn().mockResolvedValue(tiles) });
      panel.openFor(makeUnit());
      const [, , backButtonContainer] = allContainers();

      isLocked.mockReturnValue(true);
      clickPointerdown(backButtonContainer!);

      // Still open — a locked Back click must be a no-op, not close/pop the panel.
      allContainers().forEach(c => expect(c.destroy).not.toHaveBeenCalled());
    });

    it('ignores an allowed-tile click while locked', async () => {
      const target: Coordinate = { x: 1, y: 0 };
      const isLocked = vi.fn(() => false);
      const { panel, showConfirm } = makePanel({
        isLocked,
        getAllowedTiles: vi.fn().mockResolvedValue([target]),
      });
      panel.openFor(makeUnit());
      const [moveButtonContainer] = allContainers();
      clickPointerdown(moveButtonContainer!);
      await Promise.resolve();
      await Promise.resolve();
      const [overlayTileGraphics] = allGraphics();

      isLocked.mockReturnValue(true);
      clickPointerdown(overlayTileGraphics!);

      expect(showConfirm).not.toHaveBeenCalled();
    });
  });
});
