import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../../test/setup';
import {
  allGraphics,
  allTexts,
  clickPointerdown,
  firstGraphics,
  textCalls,
} from '../../test/sceneHelpers';
import { makeCfg } from '../../test/fixtures';
import { START_MATCH_BUTTON_LABEL, PANEL_BUTTON_FILL_COLOR } from '../../constants';
import type { StagePreset, GameCfg } from '../../types/api';
import type { PageBounds, SettingsPageNav } from './SettingsPage';
import StagePage from './StagePage';

function lineWidth(g: ReturnType<typeof mockScene.add.graphics>): number {
  const [width] = g.lineStyle.mock.calls[0] as [number, number, number];
  return width;
}

beforeEach(() => {
  vi.clearAllMocks();
});

const STAGE_PRESETS: StagePreset[] = [
  { name: 'Plain', description: 'A plain field.', width: 10, height: 10, maxTurns: 60 },
  { name: 'Divided', description: 'Split down the middle.', width: 12, height: 8, maxTurns: 30 },
];

function bodyBounds(): PageBounds {
  return { x: 48, y: 156, width: 1184, height: 408 };
}

function navBounds(): PageBounds {
  return { x: 48, y: 636, width: 1184, height: 108 };
}

function nav(overrides: Partial<SettingsPageNav> = {}): SettingsPageNav {
  return {
    goNext: vi.fn(),
    goBack: vi.fn(),
    startMatch: vi.fn(),
    exitToTitle: vi.fn(),
    ...overrides,
  };
}

function page(
  cfg: GameCfg = makeCfg(),
  presets: StagePreset[] = STAGE_PRESETS,
  n: SettingsPageNav = nav()
): StagePage {
  return new StagePage(cfg, presets, n);
}

describe('StagePage — header', () => {
  it('renders the "Stage Selection" title', () => {
    const p = page();

    p.renderHeaderTitle(mockScene as never, 200, 102);

    expect(mockScene.add.text).toHaveBeenCalledWith(
      200,
      102,
      'Stage Selection',
      expect.objectContaining({})
    );
  });
});

describe('StagePage — StagesPanel', () => {
  it('renders one StageCard per stagePreset, named after it', () => {
    const p = page();
    p.renderBody(mockScene as never, bodyBounds());

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Plain',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Divided',
      expect.objectContaining({})
    );
  });

  it('selects the StageCard matching gameCfg.stagePreset on mount', () => {
    const cfg = makeCfg({ stagePreset: 'Divided' });
    const p = page(cfg);
    p.renderBody(mockScene as never, bodyBounds());

    // StageCard 0 = Plain (unselected: thin border), StageCard 1 = Divided (selected: thick border).
    const [plainCardGraphics, dividedCardGraphics] = allGraphics();

    expect(lineWidth(dividedCardGraphics!)).toBeGreaterThan(lineWidth(plainCardGraphics!));
  });

  it('falls back to selecting the first StageCard when gameCfg.stagePreset matches none', () => {
    const cfg = makeCfg({ stagePreset: 'Nonexistent' });
    const p = page(cfg);
    p.renderBody(mockScene as never, bodyBounds());

    const [plainCardGraphics, dividedCardGraphics] = allGraphics();

    expect(lineWidth(plainCardGraphics!)).toBeGreaterThan(lineWidth(dividedCardGraphics!));
  });

  it('clicking a StageCard selects it, unselects the rest, and commits gameCfg immediately', () => {
    const cfg = makeCfg({ stagePreset: 'Plain' });
    const p = page(cfg);
    p.renderBody(mockScene as never, bodyBounds());

    const [, dividedCardGraphics] = allGraphics();
    clickPointerdown(dividedCardGraphics!);

    expect(cfg.stagePreset).toBe('Divided');
    expect(cfg.maxTurns).toBe(30);

    // renderBody() renders [cardG(Plain), cardG(Divided), leftArrowG, rightArrowG] (4 graphics);
    // the click re-renders both StagesPanel + StageDetailPanel, appending 4 more.
    const [, , , , plainCardGraphics2, dividedCardGraphics2] = allGraphics();
    expect(lineWidth(dividedCardGraphics2!)).toBeGreaterThan(lineWidth(plainCardGraphics2!));
  });

  it('centers the StageCard row within the StagesPanel half', () => {
    const p = page();
    const bounds = bodyBounds();
    p.renderBody(mockScene as never, bounds);

    // 2 cards x 160px + 1 gap x 12px = 332px row width, centered within (bounds.width*0.6 - 24)px
    // of content (12px padding each side of the StagesPanel's 60%-width half).
    const panelWidth = bounds.width * 0.6;
    const contentX = bounds.x + 12;
    const contentWidth = panelWidth - 24;
    const expectedX = contentX + (contentWidth - 332) / 2;

    const [firstCardGraphics] = allGraphics();
    const [x] = firstCardGraphics!.fillRoundedRect.mock.calls[0] as [number, number];
    expect(x).toBe(expectedX);
  });
});

describe('StagePage — StageDetailPanel', () => {
  it('renders the selected preset name, description, and dimensions', () => {
    const p = page();
    p.renderBody(mockScene as never, bodyBounds());

    const descriptionCall = textCalls().find(c => c[2] === 'A plain field.');
    const style = descriptionCall?.[3] as { wordWrap?: { width: number } } | undefined;
    expect(typeof style?.wordWrap?.width).toBe('number');
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '10',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'x',
      expect.objectContaining({})
    );
  });

  it("centers the 'x' glyph at InnerPanel's content center-x", () => {
    const p = page();
    const bounds = bodyBounds();
    p.renderBody(mockScene as never, bounds);

    // StageDetailPanel is the remaining 40% of the body; InnerPanel is 80% of that, centered;
    // content-x is InnerPanel's center (padding is symmetric, so it doesn't shift the center).
    const detailPanelX = bounds.x + bounds.width * 0.6;
    const detailPanelWidth = bounds.width * 0.4;
    const innerWidth = detailPanelWidth * 0.8;
    const innerX = detailPanelX + (detailPanelWidth - innerWidth) / 2;
    const expectedCx = innerX + innerWidth / 2;

    const xCall = textCalls().find(c => c[2] === 'x')!;
    expect(xCall[0]).toBe(expectedCx);
  });

  it('replaces all detail info when a different StageCard is selected', () => {
    const p = page();
    p.renderBody(mockScene as never, bodyBounds());

    const [, dividedCardGraphics] = allGraphics();
    clickPointerdown(dividedCardGraphics!);

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      'Split down the middle.',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '12',
      expect.objectContaining({})
    );
  });
});

describe('StagePage — MaxTurnsSelector', () => {
  it("seeds from the selected preset's recommended maxTurns, ignoring gameCfg.maxTurns (spec clarification)", () => {
    const cfg = makeCfg({ stagePreset: 'Plain', maxTurns: 15 });
    const p = page(cfg);
    p.renderBody(mockScene as never, bodyBounds());

    // Plain's recommended maxTurns is 60, not gameCfg.maxTurns's 15.
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '60',
      expect.objectContaining({})
    );
  });

  it('shows the 🌟 recommended glyph when the current value matches the preset default', () => {
    const p = page();
    p.renderBody(mockScene as never, bodyBounds());

    const starText = allTexts()[allTexts().length - 1]!;
    expect(starText.setVisible).toHaveBeenCalledWith(true);
  });

  it('right arrow cycles maxTurns forward and commits gameCfg.maxTurns immediately', () => {
    const cfg = makeCfg({ stagePreset: 'Plain' });
    const p = page(cfg);
    p.renderBody(mockScene as never, bodyBounds());

    // Graphics order: [cardG(Plain), cardG(Divided), leftArrowG, rightArrowG].
    const rightArrowGraphics = allGraphics()[3]!;
    clickPointerdown(rightArrowGraphics);

    // Plain's maxTurns=60 cycles forward (wraps) to 💣 (0).
    expect(cfg.maxTurns).toBe(0);
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      '💣',
      expect.objectContaining({})
    );
  });

  it('left arrow cycles maxTurns backward and commits gameCfg.maxTurns immediately', () => {
    const cfg = makeCfg({ stagePreset: 'Plain' });
    const p = page(cfg);
    p.renderBody(mockScene as never, bodyBounds());

    const leftArrowGraphics = allGraphics()[2]!;
    clickPointerdown(leftArrowGraphics);

    // Plain's maxTurns=60 cycles backward to 45.
    expect(cfg.maxTurns).toBe(45);
  });

  it('hides the 🌟 glyph once the value is cycled away from the recommended value', () => {
    const p = page(makeCfg({ stagePreset: 'Plain' }));
    p.renderBody(mockScene as never, bodyBounds());

    const leftArrowGraphics = allGraphics()[2]!;
    clickPointerdown(leftArrowGraphics);

    const starText = allTexts()[allTexts().length - 1]!;
    expect(starText.setVisible).toHaveBeenCalledWith(false);
  });

  it('re-selecting a StageCard resets MaxTurnsSelector to that preset default, discarding customization', () => {
    const cfg = makeCfg({ stagePreset: 'Plain' });
    const p = page(cfg);
    p.renderBody(mockScene as never, bodyBounds());

    clickPointerdown(allGraphics()[2]!); // left arrow -> 45
    expect(cfg.maxTurns).toBe(45);

    // Re-select Plain (StagesPanel wasn't re-rendered by the arrow click, so index 0 still
    // points at the original Plain card).
    clickPointerdown(allGraphics()[0]!);

    expect(cfg.maxTurns).toBe(60);
  });
});

describe('StagePage — StartMatchButton', () => {
  it('renders always enabled (no disabled styling) with the right label', () => {
    const p = page();
    p.renderNav(mockScene as never, navBounds());

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      START_MATCH_BUTTON_LABEL,
      expect.objectContaining({})
    );
    expect(firstGraphics().fillStyle).toHaveBeenCalledWith(
      PANEL_BUTTON_FILL_COLOR,
      expect.any(Number)
    );
  });

  it('calls nav.startMatch() when clicked', () => {
    const startMatch = vi.fn();
    const p = page(makeCfg(), STAGE_PRESETS, nav({ startMatch }));
    p.renderNav(mockScene as never, navBounds());

    clickPointerdown(firstGraphics());

    expect(startMatch).toHaveBeenCalled();
  });

  it('positions StartMatchButton flush against the NavRegion right edge', () => {
    const p = page();
    const b = navBounds();
    p.renderNav(mockScene as never, b);

    const [x] = firstGraphics().fillRoundedRect.mock.calls[0] as [number, number];
    expect(x).toBe(b.x + b.width - 144);
  });
});

describe('StagePage — BackButton delegation', () => {
  it('handleBack navigates back to UnitPage 2 (AC 15)', () => {
    const goBack = vi.fn();
    const p = page(makeCfg(), STAGE_PRESETS, nav({ goBack }));

    p.handleBack();

    expect(goBack).toHaveBeenCalled();
  });
});

describe('StagePage — destroy', () => {
  it('destroys every header/body/nav GameObject it created', () => {
    const p = page();
    p.renderHeaderTitle(mockScene as never, 200, 102);
    p.renderBody(mockScene as never, bodyBounds());
    p.renderNav(mockScene as never, navBounds());

    const graphicsAndTexts = [...allGraphics(), ...allTexts()];
    p.destroy();

    graphicsAndTexts.forEach(obj => expect(obj.destroy).toHaveBeenCalled());
  });

  it('renderNav tears down a previously rendered StartMatchButton before redrawing', () => {
    const p = page();
    p.renderNav(mockScene as never, navBounds());
    const firstButtonGraphics = firstGraphics();

    p.renderNav(mockScene as never, navBounds());

    expect(firstButtonGraphics.destroy).toHaveBeenCalled();
  });
});
