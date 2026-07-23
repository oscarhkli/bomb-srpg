import Phaser from 'phaser';
import { getCatalog, createMatchRoom, initRoom, createMatch } from '../engine/api';
import ErrorPanel from '../ui/ErrorPanel';
import { drawBackButton } from '../ui/backButton';
import UnitPage from '../ui/matchSettings/UnitPage';
import StagePage from '../ui/matchSettings/StagePage';
import type { PageBounds, SettingsPage, SettingsPageNav } from '../ui/matchSettings/SettingsPage';
import type { MatchSceneData } from './MatchScene';
import type { Catalog, GameCfg } from '../types/api';
import {
  SETTINGS_SCENE_MARGIN,
  SETTINGS_REGION_HEIGHT,
  SETTINGS_HEADER_SPACER,
  BACK_BUTTON_SIZE,
  FADE_MS,
} from '../constants';

export interface MatchSettingsSceneData {
  gameCfg?: GameCfg;
}

function defaultGameCfg(): GameCfg {
  return {
    stagePreset: 'Plain',
    p1Teams: ['King'],
    p2Teams: ['King'],
    maxTurns: 60,
    allowResetTurn: true,
  };
}

// Configures a match: 2 UnitPages (one per Player) + a StagePage, swapped via fadeTransition
// behind persistent chrome (BackButton + HeaderRegion/NavRegion).
export default class MatchSettingsScene extends Phaser.Scene {
  private gameCfg!: GameCfg;
  private pages: SettingsPage[] = [];
  private currentPageIndex = 0;
  private errorPanel!: ErrorPanel;
  // Bumped on 'shutdown' so late-resolving async work can detect a torn-down scene.
  private generation = 0;
  // Guards against a re-entrant page/match transition (double-click, rapid Back/Next).
  private isTransitioning = false;

  constructor() {
    super('MatchSettingsScene');
  }

  create(data: MatchSettingsSceneData = {}): void {
    const gen = this.generation;
    this.gameCfg = data.gameCfg ?? defaultGameCfg();
    this.pages = [];
    this.currentPageIndex = 0;
    this.isTransitioning = false;
    this.errorPanel = new ErrorPanel(this);
    this.events.once('shutdown', () => {
      this.generation++;
    });
    this.cameras.main.fadeIn(FADE_MS);

    this.renderBackButton();

    getCatalog()
      .then(catalog => {
        if (gen !== this.generation) {
          return;
        }
        // Empty archetypes or stagePresets is unplayable — report and stop.
        if (catalog.archetypes.length === 0 || catalog.stagePresets.length === 0) {
          this.errorPanel.show(
            'Failed to load match catalog: no archetypes or stage presets available'
          );
          return;
        }
        this.buildPages(catalog);
        this.showPage(0);
      })
      .catch(() => {
        if (gen !== this.generation) {
          return;
        }
        this.errorPanel.show('Failed to load match catalog');
      });
  }

  private renderBackButton(): void {
    drawBackButton(this, SETTINGS_SCENE_MARGIN, SETTINGS_SCENE_MARGIN, () => {
      this.pages[this.currentPageIndex]?.handleBack();
    });
  }

  private buildPages(catalog: Catalog): void {
    const nav: SettingsPageNav = {
      goNext: () => this.goToPage(this.currentPageIndex + 1),
      goBack: () => this.goToPage(this.currentPageIndex - 1),
      startMatch: () => this.startMatch(),
      exitToTitle: () => this.exitToTitle(),
    };
    this.pages = [
      new UnitPage(1, this.gameCfg, catalog.archetypes, nav),
      new UnitPage(2, this.gameCfg, catalog.archetypes, nav),
      new StagePage(this.gameCfg, catalog.stagePresets, nav),
    ];
  }

  private bodyBounds(): PageBounds {
    const { width, height } = this.cameras.main;
    return {
      x: SETTINGS_SCENE_MARGIN,
      y: SETTINGS_SCENE_MARGIN + SETTINGS_REGION_HEIGHT,
      width: width - SETTINGS_SCENE_MARGIN * 2,
      height: height - SETTINGS_SCENE_MARGIN * 2 - SETTINGS_REGION_HEIGHT * 2,
    };
  }

  private navBounds(): PageBounds {
    const { width, height } = this.cameras.main;
    return {
      x: SETTINGS_SCENE_MARGIN,
      y: height - SETTINGS_SCENE_MARGIN - SETTINGS_REGION_HEIGHT,
      width: width - SETTINGS_SCENE_MARGIN * 2,
      height: SETTINGS_REGION_HEIGHT,
    };
  }

  private renderActivePage(): void {
    const page = this.pages[this.currentPageIndex];
    if (!page) {
      return;
    }
    const titleX = SETTINGS_SCENE_MARGIN + BACK_BUTTON_SIZE + SETTINGS_HEADER_SPACER;
    const titleY = SETTINGS_SCENE_MARGIN + SETTINGS_REGION_HEIGHT / 2;
    page.renderHeaderTitle(this, titleX, titleY);
    page.renderBody(this, this.bodyBounds());
    page.renderNav(this, this.navBounds());
  }

  private showPage(index: number): void {
    this.currentPageIndex = index;
    this.renderActivePage();
  }

  // Fades out, then runs onFadedOut — unless the scene was torn down/recreated mid-fade
  // (generation check) or another transition is already in flight.
  private fadeOutThen(onFadedOut: () => void): void {
    if (this.isTransitioning) {
      return;
    }
    this.isTransitioning = true;
    const gen = this.generation;
    this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
    this.cameras.main.once('camerafadeoutcomplete', () => {
      if (gen !== this.generation) {
        this.isTransitioning = false;
        return;
      }
      onFadedOut();
    });
  }

  // Page swap = fadeTransition: fade out, destroy the outgoing Page, render the incoming one,
  // fade in. BackButton persists across the swap untouched.
  private goToPage(index: number): void {
    if (!this.pages[index]) {
      return;
    }
    this.fadeOutThen(() => {
      this.pages[this.currentPageIndex]?.destroy();
      this.currentPageIndex = index;
      this.renderActivePage();
      this.cameras.main.fadeIn(FADE_MS);
      this.isTransitioning = false;
    });
  }

  private exitToTitle(): void {
    this.fadeOutThen(() => {
      this.scene.start('TitleScene');
    });
  }

  private startMatch(): void {
    if (this.isTransitioning) {
      return;
    }
    this.isTransitioning = true;
    const gen = this.generation;
    this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
    const fadeDone = new Promise<void>(resolve => {
      this.cameras.main.once('camerafadeoutcomplete', () => resolve());
      // Scene may be torn down before the fade event fires; resolve anyway so this
      // promise can't hang forever (the gen check below still guards the outcome).
      this.events.once('shutdown', () => resolve());
    });
    const matchResult = createMatchRoom()
      .then(({ id }) => {
        initRoom(id);
        return createMatch({ gameCfg: this.gameCfg }).then(({ playerTokens }) => ({
          ok: true as const,
          roomId: id,
          playerTokens,
        }));
      })
      .catch((err: unknown) => ({
        ok: false as const,
        message: err instanceof Error ? err.message : String(err),
      }));

    void Promise.all([fadeDone, matchResult]).then(([, result]) => {
      if (gen !== this.generation) {
        return;
      }
      if (!result.ok) {
        this.cameras.main.fadeIn(FADE_MS);
        this.errorPanel.show(`Failed to create match: ${result.message}`);
        this.isTransitioning = false;
        return;
      }
      this.scene.start('MatchScene', {
        roomId: result.roomId,
        playerTokens: result.playerTokens,
      } satisfies MatchSceneData);
    });
  }
}
