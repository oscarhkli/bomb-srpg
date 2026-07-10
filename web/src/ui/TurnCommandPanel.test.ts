import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
  DISABLED_BUTTON_COLOR,
  GAME_FONT_FAMILY,
  ALLOWED_TILE_MOVE_COLOR,
  ALLOWED_TILE_MOVE_ALPHA,
  ALLOWED_TILE_BOMB_COLOR,
} from '../constants';
import TurnCommandPanel from './TurnCommandPanel';
import type { Coordinate, TurnCmdType, Unit } from '../types/api';

function makeUnit(overrides: Partial<Unit> = {}): Unit {
  return {
    id: 1,
    type: 'Fighter',
    position: { x: 0, y: 0 },
    speed: 2,
    bombMaxRange: 2,
    bombPower: 1,
    maxBombCount: 3,
    bombUsed: 0,
    team: 1,
    hp: 1,
    skills: [],
    hasMoved: false,
    hasUsedSkill: false,
    ...overrides,
  };
}

function allGraphics(): ReturnType<typeof mockScene.add.graphics>[] {
  return mockScene.add.graphics.mock.results.map(
    r => r.value as ReturnType<typeof mockScene.add.graphics>
  );
}

function allTexts(): ReturnType<typeof mockScene.add.text>[] {
  return mockScene.add.text.mock.results.map(r => r.value as ReturnType<typeof mockScene.add.text>);
}

function makePanel(overrides: Partial<Record<string, unknown>> = {}) {
  const defaultGetAllowedTiles = vi.fn<
    (unitId: number, turnCmdType: TurnCmdType) => Promise<Coordinate[]>
  >(() => Promise.resolve([]));
  const defaultOnError = vi.fn();
  const defaultOnConfirmedSubmit = vi.fn();
  const defaultShowConfirm = vi.fn<(onYes: () => void, onNo: () => void) => void>();
  const defaultHideConfirm = vi.fn();
  const defaultIsConfirmOpen = vi.fn(() => false);
  const callbacks = {
    getAllowedTiles: defaultGetAllowedTiles,
    onError: defaultOnError,
    onConfirmedSubmit: defaultOnConfirmedSubmit,
    showConfirm: defaultShowConfirm,
    hideConfirm: defaultHideConfirm,
    isConfirmOpen: defaultIsConfirmOpen,
    ...overrides,
  };
  const panel = new TurnCommandPanel(mockScene as never, callbacks);
  panel.setGridBounds(240, 240);
  return {
    panel,
    getAllowedTiles: callbacks.getAllowedTiles,
    onError: callbacks.onError,
    onConfirmedSubmit: callbacks.onConfirmedSubmit,
    showConfirm: callbacks.showConfirm,
    hideConfirm: callbacks.hideConfirm,
    isConfirmOpen: callbacks.isConfirmOpen,
  };
}

function latestConfirmCallbacks(showConfirm: ReturnType<typeof vi.fn>): {
  onYes: () => void;
  onNo: () => void;
} {
  const lastCall = showConfirm.mock.calls.at(-1) as [() => void, () => void];
  return { onYes: lastCall[0], onNo: lastCall[1] };
}

function clickPointerdown(g: ReturnType<typeof mockScene.add.graphics>): void {
  const onPointerDown = g.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void;
  onPointerDown();
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('TurnCommandPanel', () => {
  it('draws three pill buttons styled per spec when opened for a fresh unit', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit());

    const graphics = allGraphics();
    expect(graphics).toHaveLength(3); // moveButton, placeBombButton, backButton
    graphics.forEach(g => {
      expect(g.fillStyle).toHaveBeenCalledWith(PANEL_BUTTON_FILL_COLOR, PANEL_BUTTON_FILL_ALPHA);
      expect(g.lineStyle).toHaveBeenCalledWith(
        PANEL_BUTTON_BORDER_WIDTH,
        PANEL_BUTTON_BORDER_COLOR,
        1
      );
    });

    expect(allTexts()).toHaveLength(3);
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Move',
      expect.objectContaining({ fontFamily: GAME_FONT_FAMILY })
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Bomb',
      expect.objectContaining({ fontFamily: GAME_FONT_FAMILY })
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Back',
      expect.objectContaining({ fontFamily: GAME_FONT_FAMILY })
    );
  });

  it('renders moveButton disabled (gray, non-interactive) when unit.hasMoved is true', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit({ hasMoved: true }));

    const [moveButtonGraphics] = allGraphics();
    expect(moveButtonGraphics!.fillStyle).toHaveBeenCalledWith(
      DISABLED_BUTTON_COLOR,
      PANEL_BUTTON_FILL_ALPHA
    );
    expect(moveButtonGraphics!.lineStyle).toHaveBeenCalledWith(
      PANEL_BUTTON_BORDER_WIDTH,
      DISABLED_BUTTON_COLOR,
      1
    );
    expect(moveButtonGraphics!.setInteractive).not.toHaveBeenCalled();
  });

  it('renders placeBombButton disabled when unit.hasUsedSkill is true', () => {
    const { panel } = makePanel();

    panel.openFor(makeUnit({ hasUsedSkill: true }));

    const [, placeBombButtonGraphics] = allGraphics();
    expect(placeBombButtonGraphics!.fillStyle).toHaveBeenCalledWith(
      DISABLED_BUTTON_COLOR,
      PANEL_BUTTON_FILL_ALPHA
    );
    expect(placeBombButtonGraphics!.setInteractive).not.toHaveBeenCalled();
  });

  it('hides the panel when backButton is clicked with nothing to roll back', () => {
    const { panel } = makePanel();
    panel.openFor(makeUnit());

    const [, , backButtonGraphics] = allGraphics();
    clickPointerdown(backButtonGraphics!);

    allGraphics().forEach(g => expect(g.destroy).toHaveBeenCalled());
    allTexts().forEach(t => expect(t.destroy).toHaveBeenCalled());
  });

  it('closeImmediately destroys every panel object', () => {
    const { panel } = makePanel();
    panel.openFor(makeUnit());

    panel.closeImmediately();

    allGraphics().forEach(g => expect(g.destroy).toHaveBeenCalled());
    allTexts().forEach(t => expect(t.destroy).toHaveBeenCalled());
  });

  it('re-opening for a different unit closes the previous panel first', () => {
    const { panel } = makePanel();
    panel.openFor(makeUnit({ id: 1 }));
    const firstButtons = allGraphics();

    panel.openFor(makeUnit({ id: 2 }));

    firstButtons.forEach(g => expect(g.destroy).toHaveBeenCalled());
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

    const [moveButtonGraphics] = allGraphics();
    clickPointerdown(moveButtonGraphics!);
    await Promise.resolve();
    await Promise.resolve();

    expect(getAllowedTiles).toHaveBeenCalledWith(5, 'move');
    const overlayGraphics = allGraphics().slice(3); // after the 3 panel buttons
    expect(overlayGraphics).toHaveLength(2);
    overlayGraphics.forEach(g => {
      expect(g.fillStyle).toHaveBeenCalledWith(ALLOWED_TILE_MOVE_COLOR, ALLOWED_TILE_MOVE_ALPHA);
    });
  });

  it('backButton hides only the allowedTiles overlay, keeping the panel open, when popping past allowedTilesShown', async () => {
    const tiles: Coordinate[] = [{ x: 1, y: 0 }];
    const { panel } = makePanel({ getAllowedTiles: vi.fn().mockResolvedValue(tiles) });
    panel.openFor(makeUnit());

    const [moveButtonGraphics, , backButtonGraphics] = allGraphics();
    clickPointerdown(moveButtonGraphics!);
    await Promise.resolve();
    await Promise.resolve();

    const overlayGraphics = allGraphics().slice(3);
    expect(overlayGraphics).toHaveLength(1);

    clickPointerdown(backButtonGraphics!);

    overlayGraphics.forEach(g => expect(g.destroy).toHaveBeenCalled());
    // The panel's own 3 buttons are untouched — only the overlay was popped, not the whole panel.
    [moveButtonGraphics, backButtonGraphics].forEach(g =>
      expect(g!.destroy).not.toHaveBeenCalled()
    );
  });

  it('fetches and renders allowedTiles with placeBomb styling when placeBombButton is clicked', async () => {
    const tiles: Coordinate[] = [{ x: 1, y: 0 }];
    const { panel, getAllowedTiles } = makePanel({
      getAllowedTiles: vi.fn().mockResolvedValue(tiles),
    });
    const unit = makeUnit({ id: 5 });
    panel.openFor(unit);

    const [, placeBombButtonGraphics] = allGraphics();
    clickPointerdown(placeBombButtonGraphics!);
    await Promise.resolve();
    await Promise.resolve();

    expect(getAllowedTiles).toHaveBeenCalledWith(5, 'placeBomb');
    const [overlayGraphics] = allGraphics().slice(3);
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

    const [moveButtonGraphics] = allGraphics();
    clickPointerdown(moveButtonGraphics!);
    await Promise.resolve();
    await Promise.resolve();

    expect(onError).toHaveBeenCalledWith('network down');
    expect(allGraphics()).toHaveLength(3); // only the 3 panel buttons, no overlay
  });

  it('calls showConfirm with a target-bound onYes; invoking it calls onConfirmedSubmit', async () => {
    const target: Coordinate = { x: 1, y: 0 };
    const { panel, showConfirm, onConfirmedSubmit } = makePanel({
      getAllowedTiles: vi.fn().mockResolvedValue([target]),
    });
    const unit = makeUnit({ id: 9 });
    panel.openFor(unit);

    const [moveButtonGraphics] = allGraphics();
    clickPointerdown(moveButtonGraphics!);
    await Promise.resolve();
    await Promise.resolve();

    const [overlayTileGraphics] = allGraphics().slice(3);
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

    const [moveButtonGraphics] = allGraphics();
    clickPointerdown(moveButtonGraphics!);
    await Promise.resolve();
    await Promise.resolve();

    const [firstTile, secondTile] = allGraphics().slice(3);
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

    const [moveButtonGraphics] = allGraphics();
    clickPointerdown(moveButtonGraphics!);
    await Promise.resolve();
    await Promise.resolve();

    const [overlayTileGraphics] = allGraphics().slice(3);
    clickPointerdown(overlayTileGraphics!);

    const { onNo } = latestConfirmCallbacks(showConfirm);
    onNo();
    await Promise.resolve();
    await Promise.resolve();

    // "No" discards the recolored tile and re-renders the overlay — the caller-side cache
    // (not this class) is what makes the re-fetch a no-op network-wise in the real app.
    expect(getAllowedTiles).toHaveBeenCalledTimes(2);
    expect(overlayTileGraphics!.destroy).toHaveBeenCalled();
    const newOverlayGraphics = allGraphics().slice(4).at(-1);
    expect(newOverlayGraphics).toBeDefined();
  });
});
