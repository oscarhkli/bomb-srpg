import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mockScene } from '../test/setup';
import {
  allGraphics,
  lastGraphics,
  textCalls,
  clickPointerdown,
  pointerDownOf,
  flush,
  fireCameraFadeOutComplete,
  fireShutdown,
} from '../test/sceneHelpers';
import { makeCfg } from '../test/fixtures';
import { getCatalog, createMatchRoom, initRoom, createMatch } from '../engine/api';
import MatchSettingsScene, { type MatchSettingsSceneData } from './MatchSettingsScene';
import { NO_UNIT } from '../ui/matchSettings/formation';
import { TEAM_COLORS, NEXT_BUTTON_LABEL, START_MATCH_BUTTON_LABEL } from '../constants';
import type { Archetype, Catalog, GameCfg } from '../types/api';

vi.mock('../engine/api');

const ARCHETYPES: Archetype[] = [
  { name: 'Fighter', speed: 2, bombMaxRange: 2, skills: [] },
  { name: 'Witch', speed: 1, bombMaxRange: 3, skills: ['Fly'] },
];
const N = ARCHETYPES.length;
const STAGE_PRESETS: Catalog['stagePresets'] = [
  { name: 'Plain', description: 'A plain field', width: 5, height: 5, maxTurns: 60 },
];

function mockCatalog(overrides: Partial<Catalog> = {}): void {
  vi.mocked(getCatalog).mockResolvedValue({
    archetypes: ARCHETYPES,
    stagePresets: STAGE_PRESETS,
    ...overrides,
  });
}

async function bootScene(data: MatchSettingsSceneData = {}): Promise<MatchSettingsScene> {
  const scene = new MatchSettingsScene();
  scene.create(data);
  await flush();
  return scene;
}

// renderActivePage() creates GameObjects in this fixed order:
//   graphics: [badge, slot x5, card x N, nextButton]  (7+N total)
//   text:     [badgeLabel, title, formationHeader, slotLabel x5, (name,speed,bomb,skill) x N, nextButtonLabel]
function graphicsSince(since: number): ReturnType<typeof allGraphics> {
  return allGraphics().slice(since);
}

// The trailing N graphics a single renderActivePage() call produces, regardless of what
// rendered earlier (BackButton, prior Pages).
const PAGE_GRAPHICS_COUNT = 7 + N;
function currentPageGraphics(): ReturnType<typeof allGraphics> {
  return lastGraphics(PAGE_GRAPHICS_COUNT);
}
function slotGraphics(g: ReturnType<typeof allGraphics>, displayPos: number): (typeof g)[number] {
  return g[1 + displayPos]!;
}
function cardGraphics(g: ReturnType<typeof allGraphics>, cardIndex: number): (typeof g)[number] {
  return g[6 + cardIndex]!;
}
function nextButtonGraphics(g: ReturnType<typeof allGraphics>): (typeof g)[number] {
  return g[6 + N]!;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MatchSettingsScene — catalog loading', () => {
  it('reports an error and builds no pages when archetypes is empty (AC 3)', async () => {
    mockCatalog({ archetypes: [] });
    await bootScene();

    expect(textCalls().some(c => typeof c[2] === 'string' && c[2].includes('catalog'))).toBe(true);
    // Only the BackButton's graphics + ErrorPanel's background graphics were created — no Page.
    expect(allGraphics()).toHaveLength(2);
  });

  it('reports an error and builds no pages when stagePresets is empty (AC 3)', async () => {
    mockCatalog({ stagePresets: [] });
    await bootScene();

    expect(textCalls().some(c => typeof c[2] === 'string' && c[2].includes('catalog'))).toBe(true);
    expect(allGraphics()).toHaveLength(2);
  });

  it('reports an error when getCatalog() itself rejects', async () => {
    vi.mocked(getCatalog).mockRejectedValue(new Error('network error'));
    await bootScene();

    expect(textCalls().some(c => typeof c[2] === 'string' && c[2].includes('catalog'))).toBe(true);
  });

  it('discards a late-resolving catalog if the scene has since shut down (async-in-create guard)', async () => {
    let resolveCatalog: (catalog: Catalog) => void = () => undefined;
    vi.mocked(getCatalog).mockReturnValue(
      new Promise<Catalog>(resolve => {
        resolveCatalog = resolve;
      })
    );
    const scene = new MatchSettingsScene();
    scene.create();
    fireShutdown();

    resolveCatalog({ archetypes: ARCHETYPES, stagePresets: STAGE_PRESETS });
    await flush();

    // Only the BackButton was ever created — the resolved catalog was discarded.
    expect(allGraphics()).toHaveLength(1);
  });
});

describe('MatchSettingsScene — Scene Entry (fadeTransition)', () => {
  it('fades in on create(), completing the fadeTransition pair regardless of entry path (AC 12)', async () => {
    mockCatalog();
    await bootScene();

    expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
  });
});

describe('MatchSettingsScene — UnitPage 1 (default entry)', () => {
  it('renders the P1 TeamBadge filled Blue and a UnitCard per archetype (AC 1)', async () => {
    mockCatalog();
    await bootScene();

    const g = currentPageGraphics();
    expect(g[0]!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
    expect(textCalls().some(c => c[2] === 'P1')).toBe(true);

    for (let i = 0; i < N; i++) {
      expect(cardGraphics(g, i).fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
    }
    expect(textCalls().some(c => c[2] === 'Fighter')).toBe(true);
    expect(textCalls().some(c => c[2] === 'Witch')).toBe(true);
  });

  it('renders the carried gameCfg formation on entry (AC 4)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({
      p1Teams: ['King', 'Witch', NO_UNIT, NO_UNIT, NO_UNIT],
      p2Teams: ['King'],
    });
    await bootScene({ gameCfg });

    const g = currentPageGraphics();
    // slot array-index 1 (order number 2) is at SLOT_DISPLAY_ORDER_P1 displayPos 1.
    const slot1 = slotGraphics(g, 1);
    // Witch's icon is a 3-point triangle (drawArchetypeIcon).
    const [points] = slot1.strokePoints.mock.calls[0] as [{ x: number; y: number }[]];
    expect(points).toHaveLength(3);
  });

  it('King (order number 1, the middle slot) has no click handler and cannot be removed (AC 5)', async () => {
    mockCatalog();
    await bootScene();

    const g = currentPageGraphics();
    // King is array index 0, at SLOT_DISPLAY_ORDER_P1 displayPos 2.
    const kingSlot = slotGraphics(g, 2);
    expect(pointerDownOf(kingSlot)).toBeUndefined();
  });

  it('clicking a UnitCard places the archetype on the lowest free slot and updates gameCfg (AC 6, 9)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King'], p2Teams: ['King'] });
    await bootScene({ gameCfg });

    const g = currentPageGraphics();
    clickPointerdown(cardGraphics(g, 0)); // Fighter

    expect(gameCfg.p1Teams).toEqual(['King', 'Fighter']);
  });

  it('clicking a UnitCard does nothing when the formation is full (AC 7)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({
      p1Teams: ['King', 'Fighter', 'Witch', 'Fighter', 'Witch'],
      p2Teams: ['King'],
    });
    await bootScene({ gameCfg });

    const g = currentPageGraphics();
    clickPointerdown(cardGraphics(g, 0));

    expect(gameCfg.p1Teams).toEqual(['King', 'Fighter', 'Witch', 'Fighter', 'Witch']);
  });

  it('clicking an occupied non-King slot frees it and updates gameCfg immediately (AC 8, 9)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({
      p1Teams: ['King', 'Fighter', NO_UNIT, 'Witch', 'Witch'],
      p2Teams: ['King'],
    });
    await bootScene({ gameCfg });

    const g = currentPageGraphics();
    // Array index 1 (order number 2) is at SLOT_DISPLAY_ORDER_P1 displayPos 1.
    clickPointerdown(slotGraphics(g, 1));

    expect(gameCfg.p1Teams).toEqual(['King', NO_UNIT, NO_UNIT, 'Witch', 'Witch']);
  });

  it('NextButton renders disabled when only King is present (AC 10)', async () => {
    mockCatalog();
    await bootScene({ gameCfg: makeCfg({ p1Teams: ['King'], p2Teams: ['King'] }) });

    const g = currentPageGraphics();
    expect(pointerDownOf(nextButtonGraphics(g))).toBeUndefined();
    expect(textCalls().some(c => c[2] === NEXT_BUTTON_LABEL)).toBe(true);
  });

  it('NextButton is enabled and fadeTransitions to UnitPage 2 once 2+ slots are filled (AC 11)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King', 'Fighter'], p2Teams: ['King'] });
    await bootScene({ gameCfg });

    const g = currentPageGraphics();
    clickPointerdown(nextButtonGraphics(g));

    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);

    const sinceNav = allGraphics().length;
    fireCameraFadeOutComplete();

    const g2 = graphicsSince(sinceNav);
    expect(g2[0]!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[2]);
    expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
  });
});

describe('MatchSettingsScene — BackButton', () => {
  async function bootAndAdvanceToPage2(gameCfg: GameCfg): Promise<void> {
    await bootScene({ gameCfg });
    const g = currentPageGraphics();
    clickPointerdown(nextButtonGraphics(g));
    fireCameraFadeOutComplete();
  }

  it('fadeTransitions to TitleScene on UnitPage 1 (p3-spec011 AC 5)', async () => {
    mockCatalog();
    await bootScene();

    const backButtonGraphics = allGraphics()[0]!;
    clickPointerdown(backButtonGraphics);

    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);
    fireCameraFadeOutComplete();

    expect(mockScene.scene.start).toHaveBeenCalledWith('TitleScene');
  });

  it('returns from UnitPage 2 to UnitPage 1 (AC 14)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King', 'Fighter'], p2Teams: ['King'] });
    await bootAndAdvanceToPage2(gameCfg);
    vi.mocked(mockScene.cameras.main.fadeOut).mockClear();

    const backButtonGraphics = allGraphics()[0]!;
    const since = allGraphics().length;
    clickPointerdown(backButtonGraphics);

    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);
    fireCameraFadeOutComplete();

    const g = graphicsSince(since);
    expect(g[0]!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[1]);
  });

  it('returns from StagePage to UnitPage 2 (AC 15)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King', 'Fighter'], p2Teams: ['King', 'Witch'] });
    await bootAndAdvanceToPage2(gameCfg);

    // Now on UnitPage 2 with 2 slots filled — advance to StagePage.
    const g2 = currentPageGraphics();
    clickPointerdown(nextButtonGraphics(g2));
    fireCameraFadeOutComplete();

    expect(textCalls().some(c => c[2] === 'Stage Selection')).toBe(true);
    expect(textCalls().some(c => c[2] === START_MATCH_BUTTON_LABEL)).toBe(true);

    const backButtonGraphics = allGraphics()[0]!;
    const since = allGraphics().length;
    clickPointerdown(backButtonGraphics);
    fireCameraFadeOutComplete();

    const g3 = graphicsSince(since);
    expect(g3[0]!.fillStyle).toHaveBeenCalledWith(TEAM_COLORS[2]);
  });

  it('ignores a second NextButton click fired while the first transition is still fading (re-entrancy guard)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King', 'Fighter'], p2Teams: ['King'] });
    await bootScene({ gameCfg });
    // Isolates the transition's own fadeIn from create()'s entry fadeIn.
    vi.mocked(mockScene.cameras.main.fadeIn).mockClear();

    const g = currentPageGraphics();
    clickPointerdown(nextButtonGraphics(g));
    // Fires before camerafadeoutcomplete — the transition is still mid-fade.
    clickPointerdown(nextButtonGraphics(g));
    fireCameraFadeOutComplete();

    // Only 1 fadeIn: a second concurrent transition would have queued its own fadeOut/fadeIn pair.
    expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledTimes(1);
  });
});

describe('MatchSettingsScene — StartMatchButton', () => {
  async function bootAndAdvanceToStagePage(gameCfg: GameCfg): Promise<void> {
    await bootScene({ gameCfg });
    clickPointerdown(nextButtonGraphics(currentPageGraphics()));
    fireCameraFadeOutComplete();
    clickPointerdown(nextButtonGraphics(currentPageGraphics()));
    fireCameraFadeOutComplete();
  }

  function startMatchButtonGraphics(): ReturnType<typeof allGraphics>[number] {
    return currentPageGraphics()[currentPageGraphics().length - 1]!;
  }

  it('creates the room and match, then transitions to MatchScene with roomId + playerTokens (AC 1)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King', 'Fighter'], p2Teams: ['King', 'Witch'] });
    await bootAndAdvanceToStagePage(gameCfg);
    vi.mocked(createMatchRoom).mockResolvedValue({ id: 'room-xyz' });
    vi.mocked(createMatch).mockResolvedValue({
      success: true,
      playerTokens: ['t1', 't2'],
    });

    clickPointerdown(startMatchButtonGraphics());
    expect(mockScene.cameras.main.fadeOut).toHaveBeenCalledWith(200, 0, 0, 0);

    await flush();
    fireCameraFadeOutComplete();
    await flush();

    expect(createMatchRoom).toHaveBeenCalled();
    expect(initRoom).toHaveBeenCalledWith('room-xyz');
    expect(createMatch).toHaveBeenCalledWith({ gameCfg });
    expect(mockScene.scene.start).toHaveBeenCalledWith('MatchScene', {
      roomId: 'room-xyz',
      playerTokens: ['t1', 't2'],
    });
  });

  it('shows an error and fades back in (without transitioning) when createMatch fails', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King', 'Fighter'], p2Teams: ['King', 'Witch'] });
    await bootAndAdvanceToStagePage(gameCfg);
    vi.mocked(createMatchRoom).mockRejectedValue(new Error('network error'));

    clickPointerdown(startMatchButtonGraphics());
    await flush();
    fireCameraFadeOutComplete();
    await flush();

    expect(mockScene.scene.start).not.toHaveBeenCalled();
    expect(mockScene.cameras.main.fadeIn).toHaveBeenCalledWith(200);
    expect(textCalls().some(c => typeof c[2] === 'string' && c[2].includes('match'))).toBe(true);
  });

  it('ignores a second StartMatchButton click fired while the first is still in flight (re-entrancy guard)', async () => {
    mockCatalog();
    const gameCfg = makeCfg({ p1Teams: ['King', 'Fighter'], p2Teams: ['King', 'Witch'] });
    await bootAndAdvanceToStagePage(gameCfg);
    vi.mocked(createMatchRoom).mockResolvedValue({ id: 'room-xyz' });
    vi.mocked(createMatch).mockResolvedValue({
      success: true,
      playerTokens: ['t1', 't2'],
    });

    clickPointerdown(startMatchButtonGraphics());
    clickPointerdown(startMatchButtonGraphics());
    await flush();
    fireCameraFadeOutComplete();
    await flush();

    expect(createMatchRoom).toHaveBeenCalledTimes(1);
  });
});
