import type Phaser from 'phaser';
import type { Archetype, GameCfg } from '../../types/api';
import {
  NO_UNIT,
  SLOT_DISPLAY_ORDER_P1,
  SLOT_DISPLAY_ORDER_P2,
  deserializeTeams,
  serializeTeams,
  lowestFreeSlot,
  occupiedCount,
} from './formation';
import type { PageBounds, SettingsPage, SettingsPageNav } from './SettingsPage';
import { drawUnitSprite } from '../../rendering/boardRenderer';
import { drawPillButton, attachRectClickHandler, type PillButtonStyle } from '../pillButton';
import { colorToCss, destroyAll } from '../gameObjectUtils';
import {
  TEAM_COLORS,
  TEAM_COLOR_FALLBACK,
  GAME_FONT_FAMILY,
  SETTINGS_CORNER_RADIUS,
  SETTINGS_TEXT_FONT_SIZE,
  UNIT_PAGE_TEAM_BADGE_WIDTH,
  UNIT_PAGE_TEAM_BADGE_HEIGHT,
  UNIT_PAGE_TEAM_BADGE_CORNER_RADIUS,
  UNIT_PAGE_TITLE_GAP,
  FORMATION_PANEL_HEIGHT_RATIO,
  UNIT_FORMATION_HEADER_FONT_SIZE,
  UNIT_SLOT_SIZE,
  UNIT_SLOT_SPACING,
  UNIT_SLOT_ORDER_LABEL_INSET,
  UNIT_SLOT_ORDER_LABEL_FONT_SIZE,
  DEPTH_UNIT_SLOT_LABEL,
  UNIT_CARD_WIDTH,
  UNIT_CARD_HEIGHT,
  UNIT_CARD_PADDING,
  UNIT_CARD_SPACING,
  UNIT_CARD_SPRITE_SIZE,
  UNIT_CARD_NAME_GAP,
  UNIT_CARD_LINE_GAP,
  UNIT_CARD_STAT_GLYPH_GAP,
  ARCHETYPES_PER_ROW,
  SETTINGS_NAV_BUTTON_WIDTH,
  SETTINGS_NAV_BUTTON_HEIGHT,
  NEXT_BUTTON_LABEL,
  DISABLED_BUTTON_COLOR,
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
} from '../../constants';

type GameObj = Phaser.GameObjects.Graphics | Phaser.GameObjects.Text;

const NEXT_BUTTON_STYLE: PillButtonStyle = {
  fillColor: PANEL_BUTTON_FILL_COLOR,
  fillAlpha: PANEL_BUTTON_FILL_ALPHA,
  borderColor: PANEL_BUTTON_BORDER_COLOR,
  borderWidth: PANEL_BUTTON_BORDER_WIDTH,
};

const NEXT_BUTTON_DISABLED_STYLE: PillButtonStyle = {
  fillColor: DISABLED_BUTTON_COLOR,
  fillAlpha: PANEL_BUTTON_FILL_ALPHA,
  borderColor: DISABLED_BUTTON_COLOR,
  borderWidth: PANEL_BUTTON_BORDER_WIDTH,
};

// Builds one Player's formation (FormationPanel) from the archetypes offered by ArchetypesPanel.
// Mutates the scene's shared `gameCfg` in place on every put-on/take-off.
export default class UnitPage implements SettingsPage {
  private readonly slots: string[];
  private scene: Phaser.Scene | undefined;
  private bodyBounds: PageBounds | undefined;
  private navBounds: PageBounds | undefined;

  private headerObjects: GameObj[] = [];
  private archetypeObjects: GameObj[] = [];
  private formationObjects: GameObj[] = [];
  private navObjects: GameObj[] = [];

  constructor(
    private readonly playerIndex: 1 | 2,
    private readonly gameCfg: GameCfg,
    private readonly archetypes: Archetype[],
    private readonly nav: SettingsPageNav
  ) {
    this.slots = deserializeTeams(this.teams());
  }

  private teams(): string[] {
    return this.playerIndex === 1 ? this.gameCfg.p1Teams : this.gameCfg.p2Teams;
  }

  private teamColor(): number {
    return TEAM_COLORS[this.playerIndex] ?? TEAM_COLOR_FALLBACK;
  }

  // Team 2 faces Team 1, so its slots render in the mirrored order (formation.ts).
  private slotDisplayOrder(): readonly number[] {
    return this.playerIndex === 1 ? SLOT_DISPLAY_ORDER_P1 : SLOT_DISPLAY_ORDER_P2;
  }

  renderHeaderTitle(scene: Phaser.Scene, x: number, y: number): void {
    this.scene = scene;

    const badge = scene.add.graphics();
    badge.fillStyle(this.teamColor());
    badge.fillRoundedRect(
      x,
      y - UNIT_PAGE_TEAM_BADGE_HEIGHT / 2,
      UNIT_PAGE_TEAM_BADGE_WIDTH,
      UNIT_PAGE_TEAM_BADGE_HEIGHT,
      UNIT_PAGE_TEAM_BADGE_CORNER_RADIUS
    );
    this.headerObjects.push(badge);

    const badgeLabel = scene.add.text(
      x + UNIT_PAGE_TEAM_BADGE_WIDTH / 2,
      y,
      `P${this.playerIndex}`,
      {
        fontFamily: GAME_FONT_FAMILY,
        color: colorToCss(0xffffff),
      }
    );
    badgeLabel.setOrigin(0.5);
    this.headerObjects.push(badgeLabel);

    const title = scene.add.text(
      x + UNIT_PAGE_TEAM_BADGE_WIDTH + UNIT_PAGE_TITLE_GAP,
      y,
      'Unit Selection',
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${SETTINGS_TEXT_FONT_SIZE}px`,
        color: colorToCss(0xffffff),
      }
    );
    title.setOrigin(0, 0.5);
    this.headerObjects.push(title);
  }

  renderBody(scene: Phaser.Scene, bounds: PageBounds): void {
    this.scene = scene;
    this.bodyBounds = bounds;
    this.renderFormationPanel();
    this.renderArchetypesPanel();
  }

  renderNav(scene: Phaser.Scene, bounds: PageBounds): void {
    this.scene = scene;
    this.navBounds = bounds;
    this.renderNextButton();
  }

  // P1: no-op (TitleScene isn't reachable yet). P2: back to UnitPage 1.
  handleBack(): void {
    if (this.playerIndex === 2) {
      this.nav.goBack();
    }
  }

  destroy(): void {
    destroyAll(this.headerObjects);
    destroyAll(this.archetypeObjects);
    destroyAll(this.formationObjects);
    destroyAll(this.navObjects);
  }

  // ---- FormationPanel ----

  private renderFormationPanel(): void {
    const scene = this.scene;
    const bounds = this.bodyBounds;
    if (!scene || !bounds) {
      return;
    }
    destroyAll(this.formationObjects);

    const header = scene.add.text(bounds.x, bounds.y, 'Formation', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${UNIT_FORMATION_HEADER_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
    });
    header.setOrigin(0, 0);
    this.formationObjects.push(header);

    const rowY = bounds.y + UNIT_FORMATION_HEADER_FONT_SIZE + UNIT_SLOT_SPACING;
    const order = this.slotDisplayOrder();
    const rowWidth = order.length * UNIT_SLOT_SIZE + (order.length - 1) * UNIT_SLOT_SPACING;
    const rowStartX = bounds.x + (bounds.width - rowWidth) / 2;
    order.forEach((slotIndex, displayPos) => {
      const slotX = rowStartX + displayPos * (UNIT_SLOT_SIZE + UNIT_SLOT_SPACING);
      this.renderUnitSlot(slotIndex, slotX, rowY);
    });
  }

  private renderUnitSlot(slotIndex: number, x: number, y: number): void {
    const scene = this.scene;
    if (!scene) {
      return;
    }
    const cx = x + UNIT_SLOT_SIZE / 2;
    const cy = y + UNIT_SLOT_SIZE / 2;
    const teamColor = this.teamColor();

    const g = scene.add.graphics();
    g.fillStyle(teamColor);
    g.fillRoundedRect(x, y, UNIT_SLOT_SIZE, UNIT_SLOT_SIZE, SETTINGS_CORNER_RADIUS);
    this.formationObjects.push(g);

    const occupant = this.slots[slotIndex];
    if (occupant !== undefined && occupant !== NO_UNIT) {
      drawUnitSprite(g, cx, cy, UNIT_SLOT_SIZE, occupant, teamColor, SETTINGS_CORNER_RADIUS);
    }

    const label = scene.add.text(
      x + UNIT_SLOT_ORDER_LABEL_INSET,
      y + UNIT_SLOT_ORDER_LABEL_INSET,
      String(slotIndex + 1),
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${UNIT_SLOT_ORDER_LABEL_FONT_SIZE}px`,
        color: colorToCss(0xffffff),
      }
    );
    // The label must render above the sprite (they overlap) — the sprite (drawUnitSprite, above)
    // is left at Phaser's default depth (0).
    label.setDepth(DEPTH_UNIT_SLOT_LABEL);
    this.formationObjects.push(label);

    // King (slot 0) is fixed: no click handler, can't be removed.
    if (slotIndex === 0) {
      return;
    }
    attachRectClickHandler(g, x, y, UNIT_SLOT_SIZE, UNIT_SLOT_SIZE, () => {
      if (this.slots[slotIndex] === NO_UNIT) {
        return;
      }
      this.slots[slotIndex] = NO_UNIT;
      this.commitAndRerender();
    });
  }

  // ---- ArchetypesPanel ----
  // No scrolling yet. A static grid is rendered once; click handlers read this.slots live.

  private renderArchetypesPanel(): void {
    const scene = this.scene;
    const bounds = this.bodyBounds;
    if (!scene || !bounds) {
      return;
    }
    destroyAll(this.archetypeObjects);

    const panelY = bounds.y + bounds.height * FORMATION_PANEL_HEIGHT_RATIO;
    const rowCount = Math.ceil(this.archetypes.length / ARCHETYPES_PER_ROW);

    this.archetypes.forEach((archetype, i) => {
      const row = Math.floor(i / ARCHETYPES_PER_ROW);
      const col = i % ARCHETYPES_PER_ROW;
      // Each row is centered on its own card count, not left-stuck in a full-width grid — e.g.
      // today's 2-archetype row sits centered as its own pair.
      const cardsInRow =
        row === rowCount - 1
          ? this.archetypes.length - row * ARCHETYPES_PER_ROW
          : ARCHETYPES_PER_ROW;
      const rowWidth = cardsInRow * UNIT_CARD_WIDTH + (cardsInRow - 1) * UNIT_CARD_SPACING;
      const rowStartX = bounds.x + (bounds.width - rowWidth) / 2;
      const x = rowStartX + col * (UNIT_CARD_WIDTH + UNIT_CARD_SPACING);
      const y = panelY + row * (UNIT_CARD_HEIGHT + UNIT_CARD_SPACING);
      this.renderUnitCard(archetype, x, y);
    });
  }

  private renderUnitCard(archetype: Archetype, x: number, y: number): void {
    const scene = this.scene;
    if (!scene) {
      return;
    }
    const cx = x + UNIT_CARD_WIDTH / 2;

    const g = scene.add.graphics();
    g.fillStyle(PANEL_BUTTON_FILL_COLOR, PANEL_BUTTON_FILL_ALPHA);
    g.fillRoundedRect(x, y, UNIT_CARD_WIDTH, UNIT_CARD_HEIGHT, SETTINGS_CORNER_RADIUS);
    g.lineStyle(PANEL_BUTTON_BORDER_WIDTH, PANEL_BUTTON_BORDER_COLOR, 1);
    g.strokeRoundedRect(x, y, UNIT_CARD_WIDTH, UNIT_CARD_HEIGHT, SETTINGS_CORNER_RADIUS);
    this.archetypeObjects.push(g);

    const spriteCy = y + UNIT_CARD_PADDING + UNIT_CARD_SPRITE_SIZE / 2;
    drawUnitSprite(
      g,
      cx,
      spriteCy,
      UNIT_CARD_SPRITE_SIZE,
      archetype.name,
      this.teamColor(),
      SETTINGS_CORNER_RADIUS
    );

    const nameY = y + UNIT_CARD_PADDING + UNIT_CARD_SPRITE_SIZE + UNIT_CARD_NAME_GAP;
    const nameText = scene.add.text(cx, nameY, archetype.name, {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${SETTINGS_TEXT_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
    });
    nameText.setOrigin(0.5);
    this.archetypeObjects.push(nameText);

    // 2 separate Text objects so the gap between speed and 💣 is an exact pixel value, not a
    // literal-space approximation.
    const statsY = nameY + UNIT_CARD_LINE_GAP;
    const speedText = scene.add.text(
      cx - UNIT_CARD_STAT_GLYPH_GAP / 2,
      statsY,
      `👟 ${archetype.speed}`,
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${SETTINGS_TEXT_FONT_SIZE}px`,
        color: colorToCss(0xffffff),
      }
    );
    speedText.setOrigin(1, 0.5);
    this.archetypeObjects.push(speedText);

    const bombText = scene.add.text(
      cx + UNIT_CARD_STAT_GLYPH_GAP / 2,
      statsY,
      `💣 ${archetype.bombMaxRange}`,
      {
        fontFamily: GAME_FONT_FAMILY,
        fontSize: `${SETTINGS_TEXT_FONT_SIZE}px`,
        color: colorToCss(0xffffff),
      }
    );
    bombText.setOrigin(0, 0.5);
    this.archetypeObjects.push(bombText);

    const skillY = statsY + UNIT_CARD_LINE_GAP;
    const skillText = scene.add.text(cx, skillY, archetype.skills[0] ?? '-', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${SETTINGS_TEXT_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
    });
    skillText.setOrigin(0.5);
    this.archetypeObjects.push(skillText);

    attachRectClickHandler(g, x, y, UNIT_CARD_WIDTH, UNIT_CARD_HEIGHT, () => {
      const free = lowestFreeSlot(this.slots);
      if (free === null) {
        return;
      }
      this.slots[free] = archetype.name;
      this.commitAndRerender();
    });
  }

  // ---- NavRegion ----

  private renderNextButton(): void {
    const scene = this.scene;
    const bounds = this.navBounds;
    if (!scene || !bounds) {
      return;
    }
    destroyAll(this.navObjects);

    // Flush against the NavRegion's right edge (the region is already inset by the scene margin).
    const x = bounds.x + bounds.width - SETTINGS_NAV_BUTTON_WIDTH;
    const y = bounds.y + bounds.height / 2 - SETTINGS_NAV_BUTTON_HEIGHT / 2;
    const enabled = occupiedCount(this.slots) >= 2;

    this.navObjects.push(
      ...drawPillButton(
        scene,
        x,
        y,
        SETTINGS_NAV_BUTTON_WIDTH,
        SETTINGS_NAV_BUTTON_HEIGHT,
        NEXT_BUTTON_LABEL,
        enabled ? NEXT_BUTTON_STYLE : NEXT_BUTTON_DISABLED_STYLE,
        0,
        enabled ? () => this.nav.goNext() : undefined
      )
    );
  }

  // ---- Shared post-change flow ----

  // Only FormationPanel + NextButton depend on formation state, so only those are redrawn.
  private commitAndRerender(): void {
    if (this.playerIndex === 1) {
      this.gameCfg.p1Teams = serializeTeams(this.slots);
    } else {
      this.gameCfg.p2Teams = serializeTeams(this.slots);
    }
    this.renderFormationPanel();
    this.renderNextButton();
  }
}
