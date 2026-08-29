import Phaser from 'phaser';
import ErrorPanel from '../ui/ErrorPanel';
import { startMatch } from './startMatch';
import type { GameCfg } from '../types/api';
import {
  FADE_MS,
  GAME_FONT_FAMILY,
  TITLE_TOP_MARGIN,
  TITLE_FONT_SIZE,
  TITLE_GAME_MODE_FONT_SIZE,
  TITLE_COPYRIGHT_BOTTOM_MARGIN,
  TITLE_COPYRIGHT_FONT_SIZE,
  TITLE_HOVER_BOMB_GAP,
  TITLE_SUBMENU_LINE_GAP,
} from '../constants';

function prologueGameCfg(): GameCfg {
  return {
    vsCpu: true,
    stagePreset: 'Plain',
    p1Slots: [
      { archetype: 'King', role: 'King' },
      { archetype: 'Fighter', role: 'Normal' },
      { archetype: 'Witch', role: 'Normal' },
      { archetype: 'Fighter', role: 'Normal' },
      { archetype: 'Bandit', role: 'Normal' },
    ],
    p2Slots: [{ archetype: 'Prologue', role: 'Boss' }],
    maxTurns: 30,
    allowResetTurn: true,
  };
}

export default class TitleScene extends Phaser.Scene {
  // Guards against re-entrant option clicks once a transition has started.
  private isTransitioning = false;
  private errorPanel!: ErrorPanel;
  // Bumped on 'shutdown'; startMatch()'s async callback compares against this.
  private generation = 0;
  private rootOption!: Phaser.GameObjects.Text;
  private submenuOptions: Phaser.GameObjects.Text[] = [];
  private hoverBomb: Phaser.GameObjects.Text | undefined;

  constructor() {
    super('TitleScene');
  }

  // Registers the game font browser-wide before any Text rasterizes; canvas text never
  // triggers a lazy stylesheet font fetch, so it must be loaded explicitly.
  preload(): void {
    this.load.font('Roboto', 'fonts/roboto-400.woff2', 'woff2');
  }

  create(): void {
    this.isTransitioning = false;
    this.errorPanel = new ErrorPanel(this);
    this.events.once('shutdown', () => {
      this.generation++;
    });
    this.cameras.main.fadeIn(FADE_MS);
    this.renderTitle();
    this.renderGameModeSelectionPanel();
    this.renderCopyrightText();
  }

  private renderTitle(): void {
    const centerX = this.cameras.main.width / 2;
    const line1 = this.add.text(centerX, TITLE_TOP_MARGIN, 'Bomb', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${TITLE_FONT_SIZE}px`,
    });
    line1.setOrigin(0.5, 0);

    // Line 2's "T" starts at the x of line 1's "m" — measured via a throwaway "Bo" text.
    const prefix = this.add.text(0, 0, 'Bo', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${TITLE_FONT_SIZE}px`,
    });
    const indent = prefix.width;
    prefix.destroy();

    const line2 = this.add.text(
      centerX - line1.width / 2 + indent,
      TITLE_TOP_MARGIN + line1.height,
      'Tactics',
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${TITLE_FONT_SIZE}px`,
      }
    );
    line2.setOrigin(0, 0);
  }

  private renderGameModeSelectionPanel(): void {
    const { width, height } = this.cameras.main;
    const x = width / 2;
    const y = height / 2;

    this.rootOption = this.buildOption(x, y, 'Start Game', () => this.showSubmenu());
    this.submenuOptions = [
      this.buildOption(x, y - TITLE_SUBMENU_LINE_GAP, 'Story Mode', () =>
        this.onStoryModeClicked()
      ),
      this.buildOption(x, y, 'Battle Mode', () => this.onBattleModeClicked()),
      this.buildOption(x, y + TITLE_SUBMENU_LINE_GAP, 'Back', () => this.showRoot()),
    ];
    this.showRoot();
  }

  private buildOption(
    x: number,
    y: number,
    label: string,
    onClick: () => void
  ): Phaser.GameObjects.Text {
    const option = this.add.text(x, y, label, {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${TITLE_GAME_MODE_FONT_SIZE}px`,
    });
    option.setOrigin(0.5);
    option.setInteractive({ useHandCursor: true });
    this.attachHoverBomb(option);
    option.on('pointerdown', () => {
      this.destroyHoverBomb();
      onClick();
    });
    return option;
  }

  private attachHoverBomb(option: Phaser.GameObjects.Text): void {
    option.on('pointerover', () => {
      this.destroyHoverBomb();
      this.hoverBomb = this.add.text(
        option.x - option.width / 2 - TITLE_HOVER_BOMB_GAP,
        option.y,
        '💣',
        {
          fontFamily: GAME_FONT_FAMILY,
          fontSize: `${TITLE_GAME_MODE_FONT_SIZE}px`,
        }
      );
      this.hoverBomb.setOrigin(1, 0.5);
    });
    option.on('pointerout', () => this.destroyHoverBomb());
  }

  private destroyHoverBomb(): void {
    this.hoverBomb?.destroy();
    this.hoverBomb = undefined;
  }

  private showRoot(): void {
    if (this.isTransitioning) {
      return;
    }
    this.destroyHoverBomb();
    this.rootOption.setVisible(true);
    this.rootOption.setInteractive({ useHandCursor: true });
    this.submenuOptions.forEach(o => {
      o.setVisible(false);
      o.disableInteractive();
    });
  }

  private showSubmenu(): void {
    if (this.isTransitioning) {
      return;
    }
    this.destroyHoverBomb();
    this.rootOption.setVisible(false);
    this.rootOption.disableInteractive();
    this.submenuOptions.forEach(o => {
      o.setVisible(true);
      o.setInteractive({ useHandCursor: true });
    });
  }

  private disableSubmenuInteractive(): void {
    this.submenuOptions.forEach(o => o.disableInteractive());
  }

  private onStoryModeClicked(): void {
    this.disableSubmenuInteractive();
    startMatch({
      scene: this,
      gameCfg: prologueGameCfg(),
      isTransitioning: () => this.isTransitioning,
      setTransitioning: value => {
        this.isTransitioning = value;
        if (!value) {
          this.submenuOptions.forEach(o => o.setInteractive({ useHandCursor: true }));
        }
      },
      generation: () => this.generation,
      errorPanel: this.errorPanel,
    });
  }

  private onBattleModeClicked(): void {
    if (this.isTransitioning) {
      return;
    }
    this.isTransitioning = true;
    this.disableSubmenuInteractive();
    this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
    this.cameras.main.once('camerafadeoutcomplete', () => {
      this.scene.start('MatchSettingsScene', {});
    });
  }

  private renderCopyrightText(): void {
    const { width, height } = this.cameras.main;
    const text = this.add.text(
      width / 2,
      height - TITLE_COPYRIGHT_BOTTOM_MARGIN,
      `© ${new Date().getFullYear()} Oscar oscarhkli.com`,
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${TITLE_COPYRIGHT_FONT_SIZE}px`,
      }
    );
    text.setOrigin(0.5, 1);
  }
}
