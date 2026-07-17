import type Phaser from 'phaser';
import {
  DEPTH_TURN_COMMAND_PANEL,
  DEPTH_MATCH_SUMMARY_PANEL,
  MATCH_SUMMARY_BUTTON_SIZE,
  MATCH_SUMMARY_BUTTON_LABEL,
  MATCH_SUMMARY_BUTTON_TEXT_COLOR,
  MATCH_SUMMARY_BUTTON_ICON_FONT_SIZE,
  MATCH_SUMMARY_TEXT_COLOR,
  MATCH_SUMMARY_TEXT_FONT_SIZE,
  MATCH_SUMMARY_PANEL_WIDTH,
  MATCH_SUMMARY_PANEL_HEIGHT,
  MATCH_SUMMARY_TOP_SECTION_RATIO,
  MATCH_SUMMARY_MID_SECTION_RATIO,
  MATCH_SUMMARY_SECTION_GAP,
  LIFECYCLE_BUTTON_HEIGHT_SMALL,
  MATCH_SUMMARY_TEAM_BADGE_WIDTH,
  MATCH_SUMMARY_TEAM_BADGE_HEIGHT,
  MATCH_SUMMARY_TEAM_BADGE_CORNER_RADIUS,
  TURN_PANEL_MARGIN,
  GAME_FONT_FAMILY,
  CONFIRM_DIALOG_DIM_COLOR,
  CONFIRM_DIALOG_DIM_ALPHA,
  RESOLVE_BUTTON_LABEL,
  RESET_BUTTON_LABEL,
  SURRENDER_BUTTON_LABEL,
  BACK_BUTTON_LABEL,
  LIFECYCLE_BUTTON_WIDTH,
  PANEL_BUTTON_SPACING,
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
  DISABLED_BUTTON_COLOR,
  TEAM_COLORS,
  TEAM_COLOR_FALLBACK,
  FADE_MS,
} from '../constants';
import { destroyAll, colorToCss, fadeInTargets, fadeOutTargets } from './gameObjectUtils';
import {
  drawPillButton,
  attachRectClickHandler,
  verticalButtonY,
  type PillButtonStyle,
} from './pillButton';
import type { GameCfg, GameState } from '../types/api';

export interface MatchSummaryPanelCallbacks {
  isLocked: () => boolean;
  onButtonClicked: () => void;
  onBackButtonClicked: () => void;
  onResolveButtonClicked: () => void;
  onResetButtonClicked: () => void;
  onSurrenderButtonClicked: () => void;
}

const BUTTON_STYLE: PillButtonStyle = {
  fillColor: PANEL_BUTTON_FILL_COLOR,
  fillAlpha: PANEL_BUTTON_FILL_ALPHA,
  borderColor: PANEL_BUTTON_BORDER_COLOR,
  borderWidth: PANEL_BUTTON_BORDER_WIDTH,
};

const DISABLED_STYLE: PillButtonStyle = {
  fillColor: DISABLED_BUTTON_COLOR,
  fillAlpha: PANEL_BUTTON_FILL_ALPHA,
  borderColor: DISABLED_BUTTON_COLOR,
  borderWidth: PANEL_BUTTON_BORDER_WIDTH,
};

export default class MatchSummaryPanel {
  private buttonObjects: Phaser.GameObjects.GameObject[] = [];
  private panelObjects: (Phaser.GameObjects.Graphics | Phaser.GameObjects.Text)[] = [];

  constructor(
    private readonly scene: Phaser.Scene,
    private readonly callbacks: MatchSummaryPanelCallbacks
  ) {}

  get isOpen(): boolean {
    return this.panelObjects.length > 0;
  }

  renderButton(): void {
    destroyAll(this.buttonObjects);
    const { width } = this.scene.cameras.main;
    const x = width - TURN_PANEL_MARGIN - MATCH_SUMMARY_BUTTON_SIZE;
    const y = TURN_PANEL_MARGIN;

    const g = this.scene.add.graphics();
    g.setDepth(DEPTH_TURN_COMMAND_PANEL);
    g.setScrollFactor(0);
    g.fillRoundedRect(x, y, MATCH_SUMMARY_BUTTON_SIZE, MATCH_SUMMARY_BUTTON_SIZE, 4);

    const text = this.scene.add.text(
      x + MATCH_SUMMARY_BUTTON_SIZE / 2,
      y + MATCH_SUMMARY_BUTTON_SIZE / 2,
      MATCH_SUMMARY_BUTTON_LABEL,
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${MATCH_SUMMARY_BUTTON_ICON_FONT_SIZE}px`,
        color: colorToCss(MATCH_SUMMARY_BUTTON_TEXT_COLOR),
      }
    );
    text.setOrigin(0.5);
    text.setDepth(DEPTH_TURN_COMMAND_PANEL);
    text.setScrollFactor(0);

    attachRectClickHandler(g, x, y, MATCH_SUMMARY_BUTTON_SIZE, MATCH_SUMMARY_BUTTON_SIZE, () => {
      if (this.callbacks.isLocked()) {
        return;
      }
      this.callbacks.onButtonClicked();
    });

    this.buttonObjects = [g, text];
  }

  open(state: GameState, cfg: GameCfg): void {
    this.close();
    const { width, height } = this.scene.cameras.main;
    const x0 = width / 2 - MATCH_SUMMARY_PANEL_WIDTH / 2;
    const y0 = height / 2 - MATCH_SUMMARY_PANEL_HEIGHT / 2;
    const topHeight = MATCH_SUMMARY_PANEL_HEIGHT * MATCH_SUMMARY_TOP_SECTION_RATIO;
    const midHeight = MATCH_SUMMARY_PANEL_HEIGHT * MATCH_SUMMARY_MID_SECTION_RATIO;
    const midY0 = y0 + topHeight + MATCH_SUMMARY_SECTION_GAP;
    const panelBottomY = y0 + MATCH_SUMMARY_PANEL_HEIGHT;

    const scrim = this.scene.add.graphics();
    scrim.setDepth(DEPTH_MATCH_SUMMARY_PANEL);
    scrim.setScrollFactor(0);
    scrim.fillStyle(CONFIRM_DIALOG_DIM_COLOR, CONFIRM_DIALOG_DIM_ALPHA);
    scrim.fillRect(0, 0, width, height);
    this.panelObjects.push(scrim);

    this.renderTopSection(x0, y0, topHeight, cfg);
    this.renderMidSection(x0, midY0, midHeight, state);
    this.renderButtons(x0, panelBottomY, cfg);

    fadeInTargets(this.scene, this.panelObjects, FADE_MS, () => undefined);
  }

  close(): void {
    if (!this.isOpen) {
      return;
    }
    const objects = this.panelObjects;
    // Disable clicks immediately rather than waiting for the fade-out tween's onComplete to
    // destroy() these objects — otherwise a click during the 200ms fade can still fire a button
    // callback on a panel that's already closing.
    objects.forEach(o => o.disableInteractive());
    fadeOutTargets(this.scene, objects, FADE_MS, () => destroyAll(objects));
    this.panelObjects = [];
  }

  private addTeamBadge(x: number, y: number, teamId: number, label: string): void {
    const badge = this.scene.add.graphics();
    badge.setDepth(DEPTH_MATCH_SUMMARY_PANEL);
    badge.setScrollFactor(0);
    badge.fillStyle(TEAM_COLORS[teamId] ?? TEAM_COLOR_FALLBACK);
    badge.fillRoundedRect(
      x - MATCH_SUMMARY_TEAM_BADGE_WIDTH / 2,
      y - MATCH_SUMMARY_TEAM_BADGE_HEIGHT / 2,
      MATCH_SUMMARY_TEAM_BADGE_WIDTH,
      MATCH_SUMMARY_TEAM_BADGE_HEIGHT,
      MATCH_SUMMARY_TEAM_BADGE_CORNER_RADIUS
    );
    this.panelObjects.push(badge);
    this.addSummaryText(x, y, label);
  }

  private addSummaryText(x: number, y: number, label: string): void {
    const text = this.scene.add.text(x, y, label, {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${MATCH_SUMMARY_TEXT_FONT_SIZE}px`,
      color: colorToCss(MATCH_SUMMARY_TEXT_COLOR),
    });
    text.setOrigin(0.5);
    text.setDepth(DEPTH_MATCH_SUMMARY_PANEL);
    text.setScrollFactor(0);
    this.panelObjects.push(text);
  }

  private renderTopSection(x0: number, y0: number, sectionHeight: number, cfg: GameCfg): void {
    const labelX = x0 + MATCH_SUMMARY_PANEL_WIDTH * 0.25;
    const valueX = x0 + MATCH_SUMMARY_PANEL_WIDTH * 0.75;
    const row1Y = y0 + sectionHeight * 0.25;
    const row2Y = y0 + sectionHeight * 0.85;

    this.addSummaryText(labelX, row1Y, 'Stage');
    this.addSummaryText(valueX, row1Y, cfg.stagePreset);
    this.addSummaryText(labelX, row2Y, 'Max Turns');
    this.addSummaryText(valueX, row2Y, String(cfg.maxTurns));
  }

  private renderMidSection(x0: number, y0: number, sectionHeight: number, state: GameState): void {
    const col1X = x0 + MATCH_SUMMARY_PANEL_WIDTH * 0.17;
    const centerX = x0 + MATCH_SUMMARY_PANEL_WIDTH / 2;
    const col3X = x0 + MATCH_SUMMARY_PANEL_WIDTH * 0.83;
    const headerY = y0 + sectionHeight * 0.15;
    const row1Y = y0 + sectionHeight * 0.5;
    const row2Y = y0 + sectionHeight * 0.85;

    this.addTeamBadge(col1X, headerY, 1, 'P1');
    this.addTeamBadge(col3X, headerY, 2, 'P2');

    const team1 = this.teamSummary(state, 1);
    const team2 = this.teamSummary(state, 2);

    this.addSummaryText(col1X, row1Y, String(team1.livingUnits));
    this.addSummaryText(centerX, row1Y, 'Living Units');
    this.addSummaryText(col3X, row1Y, String(team2.livingUnits));

    this.addSummaryText(col1X, row2Y, String(team1.availableBombs));
    this.addSummaryText(centerX, row2Y, 'Available Bombs');
    this.addSummaryText(col3X, row2Y, String(team2.availableBombs));
  }

  private teamSummary(
    state: GameState,
    teamId: number
  ): { livingUnits: number; availableBombs: number } {
    const livingUnits = state.units.filter(u => u.team === teamId && u.hp > 0);
    const availableBombs = livingUnits.reduce((sum, u) => sum + (u.maxBombCount - u.bombUsed), 0);
    return { livingUnits: livingUnits.length, availableBombs };
  }

  // panelBottomY is the content box's own bottom edge (y0 + MATCH_SUMMARY_PANEL_HEIGHT) — the
  // 4-button block is bottom-aligned against it (12px gap preserved after the last button),
  // not top-anchored right below the mid-section. This leaves open space between the mid
  // section and the button block, which grows/shrinks with MATCH_SUMMARY_PANEL_HEIGHT.
  private renderButtons(x0: number, panelBottomY: number, cfg: GameCfg): void {
    const x = x0 + MATCH_SUMMARY_PANEL_WIDTH / 2 - LIFECYCLE_BUTTON_WIDTH / 2;
    const resetEnabled = cfg.allowResetTurn;

    const buttons: { label: string; style: PillButtonStyle; onClick: (() => void) | undefined }[] =
      [
        {
          label: RESOLVE_BUTTON_LABEL,
          style: BUTTON_STYLE,
          onClick: () => this.callbacks.onResolveButtonClicked(),
        },
        {
          label: RESET_BUTTON_LABEL,
          style: resetEnabled ? BUTTON_STYLE : DISABLED_STYLE,
          onClick: resetEnabled ? () => this.callbacks.onResetButtonClicked() : undefined,
        },
        {
          label: SURRENDER_BUTTON_LABEL,
          style: BUTTON_STYLE,
          onClick: () => this.callbacks.onSurrenderButtonClicked(),
        },
        {
          label: BACK_BUTTON_LABEL,
          style: BUTTON_STYLE,
          onClick: () => this.callbacks.onBackButtonClicked(),
        },
      ];

    const blockHeight =
      buttons.length * LIFECYCLE_BUTTON_HEIGHT_SMALL + (buttons.length - 1) * PANEL_BUTTON_SPACING;
    const startY = panelBottomY - PANEL_BUTTON_SPACING - blockHeight;

    buttons.forEach(({ label, style, onClick }, i) => {
      const y = verticalButtonY(startY, i, LIFECYCLE_BUTTON_HEIGHT_SMALL, PANEL_BUTTON_SPACING);
      this.panelObjects.push(
        ...drawPillButton(
          this.scene,
          x,
          y,
          LIFECYCLE_BUTTON_WIDTH,
          LIFECYCLE_BUTTON_HEIGHT_SMALL,
          label,
          style,
          DEPTH_MATCH_SUMMARY_PANEL,
          onClick,
          0
        )
      );
    });
  }
}
