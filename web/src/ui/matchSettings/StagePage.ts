import type Phaser from 'phaser';
import type { GameCfg, StagePreset } from '../../types/api';
import type { PageBounds, SettingsPage, SettingsPageNav } from './SettingsPage';
import { cycleMaxTurns, findStagePresetIndex, formatMaxTurns } from './stageSelection';
import { drawPillButton, attachRectClickHandler, type PillButtonStyle } from '../pillButton';
import { colorToCss, destroyAll, centeredRowStartX } from '../gameObjectUtils';
import {
  GAME_FONT_FAMILY,
  SETTINGS_CORNER_RADIUS,
  SETTINGS_TEXT_FONT_SIZE,
  SETTINGS_NAV_BUTTON_WIDTH,
  SETTINGS_NAV_BUTTON_HEIGHT,
  START_MATCH_BUTTON_LABEL,
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
  STAGES_PANEL_WIDTH_RATIO,
  STAGE_DETAIL_PANEL_WIDTH_RATIO,
  STAGE_PANEL_PADDING,
  STAGE_CARD_SIZE,
  STAGE_CARD_PADDING,
  STAGE_CARD_SPACING,
  STAGE_CARD_NAME_FONT_SIZE,
  STAGE_CARD_SELECTED_BORDER_WIDTH,
  STAGE_DETAIL_INNER_PANEL_SIZE_RATIO,
  STAGE_DETAIL_INNER_PANEL_PADDING,
  STAGE_DETAIL_ROW_FONT_SIZE,
  STAGE_DETAIL_ROW_GAP,
  STAGE_DETAIL_DESCRIPTION_LINES,
  STAGE_DETAIL_WIDTH_HEIGHT_GAP,
  MAX_TURNS_ARROW_LEFT_LABEL,
  MAX_TURNS_ARROW_RIGHT_LABEL,
  MAX_TURNS_ARROW_INSET,
  MAX_TURNS_RECOMMENDED_GLYPH,
  MAX_TURNS_RECOMMENDED_FONT_SIZE,
  MAX_TURNS_RECOMMENDED_GLYPH_GAP,
} from '../../constants';

type GameObj = Phaser.GameObjects.Graphics | Phaser.GameObjects.Text;

const START_MATCH_BUTTON_STYLE: PillButtonStyle = {
  fillColor: PANEL_BUTTON_FILL_COLOR,
  fillAlpha: PANEL_BUTTON_FILL_ALPHA,
  borderColor: PANEL_BUTTON_BORDER_COLOR,
  borderWidth: PANEL_BUTTON_BORDER_WIDTH,
};

// StagesPanel (StageCard grid, left half) + StageDetailPanel (InnerPanel, right half).
export default class StagePage implements SettingsPage {
  private selectedIndex: number;
  private currentMaxTurns: number;
  private scene: Phaser.Scene | undefined;
  private bodyBounds: PageBounds | undefined;
  private navBounds: PageBounds | undefined;

  private headerObjects: GameObj[] = [];
  private stagesPanelObjects: GameObj[] = [];
  private detailPanelObjects: GameObj[] = [];
  private navObjects: GameObj[] = [];

  constructor(
    private readonly gameCfg: GameCfg,
    private readonly stagePresets: StagePreset[],
    private readonly nav: SettingsPageNav
  ) {
    if (stagePresets.length === 0) {
      throw new Error('StagePage requires at least one stagePreset');
    }
    this.selectedIndex = findStagePresetIndex(stagePresets, gameCfg.stagePreset);
    this.currentMaxTurns = this.selectedPreset().maxTurns;
  }

  private selectedPreset(): StagePreset {
    const preset = this.stagePresets[this.selectedIndex];
    if (!preset) {
      throw new Error(`StagePage: no stagePreset at index ${this.selectedIndex}`);
    }
    return preset;
  }

  renderHeaderTitle(scene: Phaser.Scene, x: number, y: number): void {
    this.scene = scene;
    const title = scene.add.text(x, y, 'Stage Selection', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${SETTINGS_TEXT_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
    });
    title.setOrigin(0, 0.5);
    this.headerObjects.push(title);
  }

  renderBody(scene: Phaser.Scene, bounds: PageBounds): void {
    this.scene = scene;
    this.bodyBounds = bounds;
    this.renderStagesPanel();
    this.renderStageDetailPanel();
  }

  renderNav(scene: Phaser.Scene, bounds: PageBounds): void {
    this.scene = scene;
    this.navBounds = bounds;
    this.renderStartMatchButton();
  }

  handleBack(): void {
    this.nav.goBack();
  }

  destroy(): void {
    destroyAll(this.headerObjects);
    destroyAll(this.stagesPanelObjects);
    destroyAll(this.detailPanelObjects);
    destroyAll(this.navObjects);
  }

  // ---- StagesPanel ----

  private renderStagesPanel(): void {
    const scene = this.scene;
    const bounds = this.bodyBounds;
    if (!scene || !bounds) {
      return;
    }
    destroyAll(this.stagesPanelObjects);

    const panelWidth = bounds.width * STAGES_PANEL_WIDTH_RATIO;
    const contentX = bounds.x + STAGE_PANEL_PADDING;
    const contentWidth = panelWidth - STAGE_PANEL_PADDING * 2;
    const rowStartX = centeredRowStartX(
      contentX,
      contentWidth,
      this.stagePresets.length,
      STAGE_CARD_SIZE,
      STAGE_CARD_SPACING
    );
    const cardY = bounds.y + (bounds.height - STAGE_CARD_SIZE) / 2;

    this.stagePresets.forEach((preset, i) => {
      const x = rowStartX + i * (STAGE_CARD_SIZE + STAGE_CARD_SPACING);
      this.renderStageCard(preset, i, x, cardY);
    });
  }

  private renderStageCard(preset: StagePreset, index: number, x: number, y: number): void {
    const scene = this.scene;
    if (!scene) {
      return;
    }
    const selected = index === this.selectedIndex;

    const g = scene.add.graphics();
    g.fillStyle(PANEL_BUTTON_FILL_COLOR, PANEL_BUTTON_FILL_ALPHA);
    g.fillRoundedRect(x, y, STAGE_CARD_SIZE, STAGE_CARD_SIZE, SETTINGS_CORNER_RADIUS);
    g.lineStyle(
      selected ? STAGE_CARD_SELECTED_BORDER_WIDTH : PANEL_BUTTON_BORDER_WIDTH,
      PANEL_BUTTON_BORDER_COLOR,
      1
    );
    g.strokeRoundedRect(x, y, STAGE_CARD_SIZE, STAGE_CARD_SIZE, SETTINGS_CORNER_RADIUS);
    this.stagesPanelObjects.push(g);

    const nameText = scene.add.text(x + STAGE_CARD_SIZE / 2, y + STAGE_CARD_SIZE / 2, preset.name, {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${STAGE_CARD_NAME_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
      align: 'center',
      wordWrap: { width: STAGE_CARD_SIZE - STAGE_CARD_PADDING * 2 },
    });
    nameText.setOrigin(0.5);
    this.stagesPanelObjects.push(nameText);

    attachRectClickHandler(g, x, y, STAGE_CARD_SIZE, STAGE_CARD_SIZE, () => {
      this.selectedIndex = index;
      this.currentMaxTurns = preset.maxTurns;
      this.gameCfg.stagePreset = preset.name;
      this.gameCfg.maxTurns = preset.maxTurns;
      this.renderStagesPanel();
      this.renderStageDetailPanel();
    });
  }

  // ---- StageDetailPanel / InnerPanel ----

  private innerPanelBounds(bounds: PageBounds): PageBounds {
    const panelX = bounds.x + bounds.width * STAGES_PANEL_WIDTH_RATIO;
    const panelWidth = bounds.width * STAGE_DETAIL_PANEL_WIDTH_RATIO;
    const innerWidth = panelWidth * STAGE_DETAIL_INNER_PANEL_SIZE_RATIO;
    const innerHeight = bounds.height * STAGE_DETAIL_INNER_PANEL_SIZE_RATIO;
    return {
      x: panelX + (panelWidth - innerWidth) / 2,
      y: bounds.y + (bounds.height - innerHeight) / 2,
      width: innerWidth,
      height: innerHeight,
    };
  }

  // Adds a detailPanelObjects Text at the shared 24px/white base style, with per-call overrides.
  private addDetailText(
    scene: Phaser.Scene,
    x: number,
    y: number,
    content: string,
    style?: Partial<Phaser.Types.GameObjects.Text.TextStyle>
  ): Phaser.GameObjects.Text {
    const text = scene.add.text(x, y, content, {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${STAGE_DETAIL_ROW_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
      ...style,
    });
    this.detailPanelObjects.push(text);
    return text;
  }

  private renderStageDetailPanel(): void {
    const scene = this.scene;
    const bounds = this.bodyBounds;
    if (!scene || !bounds) {
      return;
    }
    destroyAll(this.detailPanelObjects);

    const preset = this.selectedPreset();
    const inner = this.innerPanelBounds(bounds);
    const contentX = inner.x + STAGE_DETAIL_INNER_PANEL_PADDING;
    const contentWidth = inner.width - STAGE_DETAIL_INNER_PANEL_PADDING * 2;
    const cx = contentX + contentWidth / 2;

    let rowY = inner.y + STAGE_DETAIL_INNER_PANEL_PADDING;

    const nameText = this.addDetailText(scene, cx, rowY, preset.name);
    nameText.setOrigin(0.5, 0);
    rowY += STAGE_DETAIL_ROW_FONT_SIZE + STAGE_DETAIL_ROW_GAP;

    const descriptionText = this.addDetailText(scene, cx, rowY, preset.description, {
      align: 'center',
      wordWrap: { width: contentWidth },
      maxLines: STAGE_DETAIL_DESCRIPTION_LINES,
    });
    descriptionText.setOrigin(0.5, 0);
    rowY += STAGE_DETAIL_ROW_FONT_SIZE * STAGE_DETAIL_DESCRIPTION_LINES + STAGE_DETAIL_ROW_GAP;

    const xText = this.addDetailText(scene, cx, rowY, 'x');
    xText.setOrigin(0.5, 0);

    const widthText = this.addDetailText(
      scene,
      cx - STAGE_DETAIL_WIDTH_HEIGHT_GAP,
      rowY,
      `${preset.width}`
    );
    widthText.setOrigin(1, 0);

    const heightText = this.addDetailText(
      scene,
      cx + STAGE_DETAIL_WIDTH_HEIGHT_GAP,
      rowY,
      `${preset.height}`
    );
    heightText.setOrigin(0, 0);
    rowY += STAGE_DETAIL_ROW_FONT_SIZE + STAGE_DETAIL_ROW_GAP;

    this.renderMaxTurnsSelector(inner, rowY, cx);
  }

  // Inset is measured from InnerPanel's own edge, not the padded content edge.
  private renderMaxTurnsSelector(inner: PageBounds, rowY: number, cx: number): void {
    const scene = this.scene;
    if (!scene) {
      return;
    }

    this.renderMaxTurnsArrow(
      inner.x + MAX_TURNS_ARROW_INSET,
      rowY,
      MAX_TURNS_ARROW_LEFT_LABEL,
      () => this.onCycleMaxTurns(-1)
    );
    this.renderMaxTurnsArrow(
      inner.x + inner.width - MAX_TURNS_ARROW_INSET,
      rowY,
      MAX_TURNS_ARROW_RIGHT_LABEL,
      () => this.onCycleMaxTurns(1)
    );

    const valueText = this.addDetailText(scene, cx, rowY, formatMaxTurns(this.currentMaxTurns));
    valueText.setOrigin(0.5, 0);

    const isRecommended = this.currentMaxTurns === this.selectedPreset().maxTurns;
    const starText = this.addDetailText(
      scene,
      cx + MAX_TURNS_RECOMMENDED_GLYPH_GAP,
      rowY,
      MAX_TURNS_RECOMMENDED_GLYPH,
      { fontSize: `${MAX_TURNS_RECOMMENDED_FONT_SIZE}px` }
    );
    starText.setOrigin(0, 0);
    starText.setVisible(isRecommended);
  }

  private renderMaxTurnsArrow(cx: number, y: number, label: string, onClick: () => void): void {
    const scene = this.scene;
    if (!scene) {
      return;
    }
    const hitSize = STAGE_DETAIL_ROW_FONT_SIZE;
    const g = scene.add.graphics();
    this.detailPanelObjects.push(g);
    attachRectClickHandler(g, cx - hitSize / 2, y, hitSize, hitSize, onClick);

    const text = this.addDetailText(scene, cx, y, label, {
      color: colorToCss(PANEL_BUTTON_BORDER_COLOR),
    });
    text.setOrigin(0.5, 0);
  }

  private onCycleMaxTurns(delta: 1 | -1): void {
    this.currentMaxTurns = cycleMaxTurns(this.currentMaxTurns, delta);
    this.gameCfg.maxTurns = this.currentMaxTurns;
    this.renderStageDetailPanel();
  }

  // ---- NavRegion ----

  private renderStartMatchButton(): void {
    const scene = this.scene;
    const bounds = this.navBounds;
    if (!scene || !bounds) {
      return;
    }
    destroyAll(this.navObjects);

    const x = bounds.x + bounds.width - SETTINGS_NAV_BUTTON_WIDTH;
    const y = bounds.y + bounds.height / 2 - SETTINGS_NAV_BUTTON_HEIGHT / 2;
    this.navObjects.push(
      ...drawPillButton(
        scene,
        x,
        y,
        SETTINGS_NAV_BUTTON_WIDTH,
        SETTINGS_NAV_BUTTON_HEIGHT,
        START_MATCH_BUTTON_LABEL,
        START_MATCH_BUTTON_STYLE,
        0,
        () => this.nav.startMatch()
      )
    );
  }
}
