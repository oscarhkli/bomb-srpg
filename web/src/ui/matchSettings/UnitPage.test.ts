import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../../test/setup';
import { allGraphics, allTexts, clickPointerdown, pointerDownOf } from '../../test/sceneHelpers';
import { makeCfg } from '../../test/fixtures';
import { TEAM_COLORS, NEXT_BUTTON_LABEL, DISABLED_BUTTON_COLOR } from '../../constants';
import type { Archetype, GameCfg } from '../../types/api';
import type { PageBounds, SettingsPageNav } from './SettingsPage';
import { NO_UNIT } from './formation';
import UnitPage from './UnitPage';

beforeEach(() => {
  vi.clearAllMocks();
});

const ARCHETYPES: Archetype[] = [
  { name: 'Fighter', speed: 2, bombMaxRange: 2, skills: [] },
  { name: 'Witch', speed: 1, bombMaxRange: 3, skills: ['Fly'] },
];

function bodyBounds(): PageBounds {
  return { x: 48, y: 156, width: 1184, height: 408 };
}

function navBounds(): PageBounds {
  return { x: 48, y: 636, width: 1184, height: 108 };
}

function nav(overrides: Partial<SettingsPageNav> = {}): SettingsPageNav {
  return { goNext: vi.fn(), goBack: vi.fn(), startMatch: vi.fn(), ...overrides };
}

function page(playerIndex: 1 | 2, cfg: GameCfg, n: SettingsPageNav = nav()): UnitPage {
  return new UnitPage(playerIndex, cfg, ARCHETYPES, n);
}

describe('UnitPage — header', () => {
  it('fills the TeamBadge with Blue for Player 1', () => {
    const p = page(1, makeCfg());
    p.renderHeaderTitle(mockScene as never, 200, 102);

    expect(allGraphics()[0]!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
  });

  it('fills the TeamBadge with Red for Player 2', () => {
    const p = page(2, makeCfg());
    p.renderHeaderTitle(mockScene as never, 200, 102);

    expect(allGraphics()[0]!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[2]);
  });

  it('titles the page "[P{X}] Unit Selection"', () => {
    const p = page(2, makeCfg());
    p.renderHeaderTitle(mockScene as never, 200, 102);

    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      102,
      'P2',
      expect.objectContaining({})
    );
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      102,
      'Unit Selection',
      expect.objectContaining({})
    );
  });
});

describe('UnitPage — FormationPanel', () => {
  it('renders King in the middle slot (display position 2) with no click handler (AC 5)', () => {
    const p = page(1, makeCfg({ p1Teams: ['King'] }));
    p.renderBody(mockScene as never, bodyBounds());

    // Graphics order: [formationHeader is text, slot(0)..slot(4) as graphics in display order]
    // SLOT_DISPLAY_ORDER_P1 = [3, 1, 0, 2, 4] -> displayPos 2 renders slotIndex 0 (King).
    const slotGraphics = allGraphics()[2]!;
    expect(pointerDownOf(slotGraphics)).toBeUndefined();
  });

  it('places a clicked UnitCard on the lowest free slot (AC 6, 9)', () => {
    const cfg = makeCfg({ p1Teams: ['King'] });
    const p = page(1, cfg);
    p.renderBody(mockScene as never, bodyBounds());

    // 5 slot graphics + N card graphics follow the formation header text.
    const cardGraphics = allGraphics()[5]!; // first UnitCard (Fighter)
    clickPointerdown(cardGraphics);

    expect(cfg.p1Teams).toEqual(['King', 'Fighter']);
  });

  it('does nothing when a UnitCard is clicked and every slot is full (AC 7)', () => {
    const cfg = makeCfg({ p1Teams: ['King', 'Fighter', 'Witch', 'Witch', 'Fighter'] });
    const p = page(1, cfg);
    p.renderBody(mockScene as never, bodyBounds());

    const cardGraphics = allGraphics()[5]!;
    clickPointerdown(cardGraphics);

    expect(cfg.p1Teams).toEqual(['King', 'Fighter', 'Witch', 'Witch', 'Fighter']);
  });

  it('frees a clicked non-King UnitSlot (AC 8)', () => {
    const cfg = makeCfg({ p1Teams: ['King', 'Fighter'] });
    const p = page(1, cfg);
    p.renderBody(mockScene as never, bodyBounds());

    // SLOT_DISPLAY_ORDER_P1 = [3, 1, 0, 2, 4]; displayPos 1 renders slotIndex 1 (Fighter).
    const slotIndex1Graphics = allGraphics()[1]!;
    clickPointerdown(slotIndex1Graphics);

    expect(cfg.p1Teams).toEqual(['King']);
  });

  it('leaves non-contiguous gaps in gameCfg after put-on/take-off (AC 9)', () => {
    const cfg = makeCfg({ p1Teams: ['King', 'Fighter', 'Witch', 'Witch', 'Fighter'] });
    const p = page(1, cfg);
    p.renderBody(mockScene as never, bodyBounds());

    // Graphics are created in display-position order (SLOT_DISPLAY_ORDER_P1 = [3, 1, 0, 2, 4]):
    // index 1 -> slotIndex 1 (Fighter), index 3 -> slotIndex 2 (Witch).
    clickPointerdown(allGraphics()[1]!);
    p.renderBody(mockScene as never, bodyBounds());
    clickPointerdown(allGraphics()[3]!);

    expect(cfg.p1Teams).toEqual(['King', NO_UNIT, NO_UNIT, 'Witch', 'Fighter']);
  });

  it('frees the correct non-King UnitSlot for Player 2 (mirrored SLOT_DISPLAY_ORDER_P2)', () => {
    const cfg = makeCfg({ p2Teams: ['King', 'Fighter', 'Witch'] });
    const p = page(2, cfg);
    p.renderBody(mockScene as never, bodyBounds());

    // SLOT_DISPLAY_ORDER_P2 = [4, 2, 0, 1, 3]; displayPos 1 renders slotIndex 2 (Witch), unlike
    // P1 where displayPos 1 renders slotIndex 1.
    clickPointerdown(allGraphics()[1]!);

    expect(cfg.p2Teams).toEqual(['King', 'Fighter']);
  });
});

describe('UnitPage — Panel stacking', () => {
  it('renders ArchetypesPanel below FormationPanel (not a side-by-side column)', () => {
    const p = page(1, makeCfg({ p1Teams: ['King'] }));
    const bounds = bodyBounds();
    p.renderBody(mockScene as never, bounds);

    // Graphics order: slot(0)..slot(4) (5 graphics), then the first UnitCard (Fighter).
    const firstCardGraphics = allGraphics()[5]!;
    const [, cardY] = firstCardGraphics.fillRoundedRect.mock.calls[0] as [number, number];

    expect(cardY).toBeGreaterThanOrEqual(bounds.y + bounds.height * 0.35);
  });

  it('centers the FormationPanel UnitSlot row within the panel width', () => {
    const p = page(1, makeCfg({ p1Teams: ['King'] }));
    const bounds = bodyBounds();
    p.renderBody(mockScene as never, bounds);

    // 5 slots x 96px + 4 gaps x 12px = 528px row width, centered in a 1184px-wide panel.
    const firstSlotGraphics = allGraphics()[0]!;
    const [slotX] = firstSlotGraphics.fillRoundedRect.mock.calls[0] as [number, number];

    expect(slotX).toBe(bounds.x + (bounds.width - 528) / 2);
  });

  it('centers a partial ArchetypesPanel row on its own card count, not a full 4-column block', () => {
    const p = page(1, makeCfg({ p1Teams: ['King'] }));
    const bounds = bodyBounds();
    p.renderBody(mockScene as never, bounds);

    // 2 archetypes x 180px + 1 gap x 12px = 372px row width, centered as its own pair.
    const firstCardGraphics = allGraphics()[5]!;
    const [cardX] = firstCardGraphics.fillRoundedRect.mock.calls[0] as [number, number];

    expect(cardX).toBe(bounds.x + (bounds.width - 372) / 2);
  });
});

describe('UnitPage — NextButton', () => {
  it('renders as DisabledButton when only King is in FormationPanel (AC 10)', () => {
    const p = page(1, makeCfg({ p1Teams: ['King'] }));
    p.renderNav(mockScene as never, navBounds());

    const nextButtonGraphics = allGraphics()[0]!;
    expect(nextButtonGraphics.fillStyle).toHaveBeenCalledWith(
      DISABLED_BUTTON_COLOR,
      expect.any(Number)
    );
    expect(pointerDownOf(nextButtonGraphics)).toBeUndefined();
  });

  it('is clickable and calls nav.goNext with >= 2 units in FormationPanel (AC 11)', () => {
    const goNext = vi.fn();
    const p = page(1, makeCfg({ p1Teams: ['King', 'Fighter'] }), nav({ goNext }));
    p.renderNav(mockScene as never, navBounds());

    clickPointerdown(allGraphics()[0]!);

    expect(goNext).toHaveBeenCalled();
    expect(mockScene.add.text).toHaveBeenCalledWith(
      expect.any(Number),
      expect.any(Number),
      NEXT_BUTTON_LABEL,
      expect.objectContaining({})
    );
  });

  it('positions NextButton flush against the NavRegion right edge', () => {
    const p = page(1, makeCfg({ p1Teams: ['King', 'Fighter'] }));
    const bounds = navBounds();
    p.renderNav(mockScene as never, bounds);

    const nextButtonGraphics = allGraphics()[0]!;
    const [x] = nextButtonGraphics.fillRoundedRect.mock.calls[0] as [number, number];

    expect(x).toBe(bounds.x + bounds.width - 144);
  });

  it('re-renders (enabling) NextButton after a put-on crosses the 2-unit threshold', () => {
    const goNext = vi.fn();
    const cfg = makeCfg({ p1Teams: ['King'] });
    const p = page(1, cfg, nav({ goNext }));
    p.renderNav(mockScene as never, navBounds());
    p.renderBody(mockScene as never, bodyBounds());

    // renderBody's card graphics come after renderNav's + formation's own graphics; click the
    // first UnitCard to cross the King-only -> King+1 threshold.
    const allG = allGraphics();
    const firstCardGraphics = allG[allG.length - ARCHETYPES.length]!;
    clickPointerdown(firstCardGraphics);

    const rerenderedNextButton = allGraphics()[allGraphics().length - 1]!;
    clickPointerdown(rerenderedNextButton);
    expect(goNext).toHaveBeenCalled();
  });
});

describe('UnitPage — BackButton delegation', () => {
  it('Player 1: handleBack is a no-op (AC 13)', () => {
    const goBack = vi.fn();
    const p = page(1, makeCfg(), nav({ goBack }));

    p.handleBack();

    expect(goBack).not.toHaveBeenCalled();
  });

  it('Player 2: handleBack navigates to UnitPage 1 (AC 14)', () => {
    const goBack = vi.fn();
    const p = page(2, makeCfg(), nav({ goBack }));

    p.handleBack();

    expect(goBack).toHaveBeenCalled();
  });
});

describe('UnitPage — destroy', () => {
  it('destroys every header/body/nav GameObject it created', () => {
    const p = page(1, makeCfg({ p1Teams: ['King', 'Fighter'] }));
    p.renderHeaderTitle(mockScene as never, 200, 102);
    p.renderBody(mockScene as never, bodyBounds());
    p.renderNav(mockScene as never, navBounds());

    const graphicsAndTexts = [...allGraphics(), ...allTexts()];
    p.destroy();

    graphicsAndTexts.forEach(obj => expect(obj.destroy).toHaveBeenCalled());
  });
});
