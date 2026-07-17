import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  firstGraphics,
  allGraphics,
  clickPointerdown,
  allTexts,
  textCalls,
  tweenConfigAt,
} from '../test/sceneHelpers';
import { makeState, makeCfg, makeUnit } from '../test/fixtures';
import {
  DEPTH_TURN_COMMAND_PANEL,
  DEPTH_MATCH_SUMMARY_PANEL,
  MATCH_SUMMARY_BUTTON_SIZE,
  MATCH_SUMMARY_BUTTON_LABEL,
  MATCH_SUMMARY_TEXT_FONT_SIZE,
  LIFECYCLE_BUTTON_HEIGHT_SMALL,
  MATCH_SUMMARY_PANEL_HEIGHT,
  TEAM_COLORS,
  TURN_PANEL_MARGIN,
  CONFIRM_DIALOG_DIM_COLOR,
  CONFIRM_DIALOG_DIM_ALPHA,
  RESET_BUTTON_LABEL,
  SURRENDER_BUTTON_LABEL,
  BACK_BUTTON_LABEL,
  RESOLVE_BUTTON_LABEL,
  DISABLED_BUTTON_COLOR,
} from '../constants';
import MatchSummaryPanel from './MatchSummaryPanel';

function makePanel(overrides: Partial<Record<string, unknown>> = {}) {
  const callbacks = {
    isLocked: vi.fn(() => false),
    onButtonClicked: vi.fn(),
    onBackButtonClicked: vi.fn(),
    onResolveButtonClicked: vi.fn(),
    onResetButtonClicked: vi.fn(),
    onSurrenderButtonClicked: vi.fn(),
    ...overrides,
  };
  const panel = new MatchSummaryPanel(mockScene as never, callbacks);
  return { panel, ...callbacks };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MatchSummaryPanel button', () => {
  it('draws a 48x48 rounded square at the top-right, mirroring TurnPanel margin, at DEPTH_TURN_COMMAND_PANEL', () => {
    const { panel } = makePanel();

    panel.renderButton();

    const button = firstGraphics();
    // Camera width is 1280 in the test mock (see test/setup.ts).
    const expectedX = 1280 - TURN_PANEL_MARGIN - MATCH_SUMMARY_BUTTON_SIZE;
    expect(button.fillRoundedRect).toHaveBeenCalledWith(
      expectedX,
      TURN_PANEL_MARGIN,
      MATCH_SUMMARY_BUTTON_SIZE,
      MATCH_SUMMARY_BUTTON_SIZE,
      expect.any(Number)
    );
    expect(button.setDepth).toHaveBeenCalledWith(DEPTH_TURN_COMMAND_PANEL);
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      MATCH_SUMMARY_BUTTON_LABEL,
      expect.objectContaining({})
    );
  });

  it('calls onButtonClicked when clicked while unlocked', () => {
    const { panel, onButtonClicked } = makePanel();
    panel.renderButton();

    clickPointerdown(firstGraphics());

    expect(onButtonClicked).toHaveBeenCalledOnce();
  });

  it('is a no-op when clicked while locked', () => {
    const { panel, onButtonClicked } = makePanel({ isLocked: vi.fn(() => true) });
    panel.renderButton();

    clickPointerdown(firstGraphics());

    expect(onButtonClicked).not.toHaveBeenCalled();
  });
});

describe('MatchSummaryPanel.open', () => {
  it('fades in a full-canvas scrim at DEPTH_MATCH_SUMMARY_PANEL, consistent with ConfirmDialog dim styling', () => {
    const { panel } = makePanel();

    panel.open(makeState(), makeCfg());

    const scrim = firstGraphics();
    expect(scrim.fillStyle).toHaveBeenCalledWith(
      CONFIRM_DIALOG_DIM_COLOR,
      CONFIRM_DIALOG_DIM_ALPHA
    );
    expect(scrim.fillRect).toHaveBeenCalledWith(0, 0, 1280, 720);
    expect(scrim.setDepth).toHaveBeenCalledWith(DEPTH_MATCH_SUMMARY_PANEL);
    expect(scrim.setScrollFactor).toHaveBeenCalledWith(0);
    const tweenCall = tweenConfigAt(0) as { alpha: number; duration: number };
    expect(tweenCall.alpha).toBe(1);
    expect(tweenCall.duration).toBe(200);
  });

  it('renders gameCfg.stagePreset and gameCfg.maxTurns in the top section', () => {
    const { panel } = makePanel();

    panel.open(makeState(), makeCfg({ stagePreset: 'MAP01', maxTurns: 30 }));

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'MAP01',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '30',
      expect.objectContaining({})
    );
  });

  it('computes living units and available bombs per team from state.units', () => {
    const { panel } = makePanel();
    const state = makeState({
      units: [
        makeUnit({ id: 1, team: 1, hp: 1, maxBombCount: 3, bombUsed: 1 }),
        makeUnit({ id: 2, team: 1, hp: 0, maxBombCount: 3, bombUsed: 0 }), // dead — excluded
        makeUnit({ id: 3, team: 2, hp: 1, maxBombCount: 2, bombUsed: 0 }),
      ],
    });

    panel.open(state, makeCfg());

    // Team 1: 1 living unit, 2 available bombs (3 - 1). Team 2: 1 living unit, 2 available bombs.
    const textValues = textCalls().map(c => c[2]);
    expect(textValues).toContain('1'); // living units, both teams
    expect(textValues).toContain('2'); // available bombs, both teams
  });

  it('renders the 4 buttons: Resolve, Reset, Surrender, Back', () => {
    const { panel } = makePanel();

    panel.open(makeState(), makeCfg());

    for (const label of [
      RESOLVE_BUTTON_LABEL,
      RESET_BUTTON_LABEL,
      SURRENDER_BUTTON_LABEL,
      BACK_BUTTON_LABEL,
    ]) {
      expect(mockScene.add.text).toHaveBeenCalledWith(
        expect.any(Number),
        expect.any(Number),
        label,
        expect.objectContaining({})
      );
    }
  });

  it('disables ResetTurnButton (no click handler, disabled color) when gameCfg.allowResetTurn is false', () => {
    const { panel } = makePanel();

    panel.open(makeState(), makeCfg({ allowResetTurn: false }));

    // Last 4 graphics are always [Resolve, Reset, Surrender, Back], regardless of how many
    // graphics (scrim, P1/P2 team badges) precede them.
    const [, resetButtonGraphics] = allGraphics().slice(-4);
    expect(resetButtonGraphics!.fillStyle).toHaveBeenCalledWith(
      DISABLED_BUTTON_COLOR,
      expect.any(Number)
    );
    expect(resetButtonGraphics!.setInteractive).not.toHaveBeenCalled();
  });

  it('invokes onResolveButtonClicked/onResetButtonClicked/onSurrenderButtonClicked/onBackButtonClicked when their buttons are clicked', () => {
    const {
      panel,
      onResolveButtonClicked,
      onResetButtonClicked,
      onSurrenderButtonClicked,
      onBackButtonClicked,
    } = makePanel();

    panel.open(makeState(), makeCfg());

    // Last 4 graphics are always [Resolve, Reset, Surrender, Back].
    const [resolveG, resetG, surrenderG, backG] = allGraphics().slice(-4);
    clickPointerdown(resolveG!);
    expect(onResolveButtonClicked).toHaveBeenCalledOnce();

    clickPointerdown(resetG!);
    expect(onResetButtonClicked).toHaveBeenCalledOnce();

    clickPointerdown(surrenderG!);
    expect(onSurrenderButtonClicked).toHaveBeenCalledOnce();

    clickPointerdown(backG!);
    expect(onBackButtonClicked).toHaveBeenCalledOnce();
  });

  it('isOpen reflects whether the panel is currently shown', () => {
    const { panel } = makePanel();

    expect(panel.isOpen).toBe(false);
    panel.open(makeState(), makeCfg());
    expect(panel.isOpen).toBe(true);
  });

  it('renders body text at MATCH_SUMMARY_TEXT_FONT_SIZE (36px)', () => {
    const { panel } = makePanel();

    panel.open(makeState(), makeCfg({ stagePreset: 'MAP01' }));

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'MAP01',
      expect.objectContaining({ fontSize: `${MATCH_SUMMARY_TEXT_FONT_SIZE}px` })
    );
  });

  it('wraps P1 and P2 in a rounded square filled with their TEAM_COLORS entry', () => {
    const { panel } = makePanel();

    panel.open(makeState(), makeCfg());

    // First two graphics after the scrim are the P1/P2 badges (drawn before any button).
    const [, p1Badge, p2Badge] = allGraphics();
    expect(p1Badge!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
    expect(p1Badge!.fillRoundedRect).toHaveBeenCalled();
    expect(p2Badge!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[2]);
    expect(p2Badge!.fillRoundedRect).toHaveBeenCalled();
  });

  it('resizes the 4 lifecycle buttons to LIFECYCLE_BUTTON_HEIGHT_SMALL (60% height) and bottom-aligns them in the content box', () => {
    const { panel } = makePanel();

    panel.open(makeState(), makeCfg());

    const [resolveG, , , backG] = allGraphics().slice(-4);
    const resolveCall = resolveG!.fillRoundedRect.mock.calls[0]!;
    const backCall = backG!.fillRoundedRect.mock.calls[0]!;
    expect(resolveCall[3]).toBe(LIFECYCLE_BUTTON_HEIGHT_SMALL);

    // Camera height is 720; the content box is vertically centered around it. The Back button
    // (bottom-most) should end 12px above the box's bottom edge (y0 + MATCH_SUMMARY_PANEL_HEIGHT).
    const y0 = (720 - MATCH_SUMMARY_PANEL_HEIGHT) / 2;
    const boxBottom = y0 + MATCH_SUMMARY_PANEL_HEIGHT;
    const backBottomEdge = (backCall[1] as number) + LIFECYCLE_BUTTON_HEIGHT_SMALL;
    expect(backBottomEdge).toBeCloseTo(boxBottom - 12, 5);
  });
});

describe('MatchSummaryPanel.close', () => {
  it('fades out then destroys all panel objects', () => {
    const { panel } = makePanel();
    panel.open(makeState(), makeCfg());
    const objectsBeforeClose = [...allTexts()];

    panel.close();
    const fadeOutCall = tweenConfigAt(1) as { onComplete?: () => void; alpha: number };
    expect(fadeOutCall.alpha).toBe(0);
    objectsBeforeClose.forEach(o => expect(o.destroy).not.toHaveBeenCalled());

    fadeOutCall.onComplete?.();

    objectsBeforeClose.forEach(o => expect(o.destroy).toHaveBeenCalled());
    expect(panel.isOpen).toBe(false);
  });
});
