import type Phaser from 'phaser';
import type { PageBounds, SettingsPage, SettingsPageNav } from './SettingsPage';
import { drawPillButton, type PillButtonStyle } from '../pillButton';
import { colorToCss, destroyAll } from '../gameObjectUtils';
import {
  GAME_FONT_FAMILY,
  SETTINGS_TEXT_FONT_SIZE,
  SETTINGS_NAV_BUTTON_WIDTH,
  SETTINGS_NAV_BUTTON_HEIGHT,
  START_MATCH_BUTTON_LABEL,
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
} from '../../constants';

type GameObj = Phaser.GameObjects.Graphics | Phaser.GameObjects.Text;

const START_MATCH_BUTTON_STYLE: PillButtonStyle = {
  fillColor: PANEL_BUTTON_FILL_COLOR,
  fillAlpha: PANEL_BUTTON_FILL_ALPHA,
  borderColor: PANEL_BUTTON_BORDER_COLOR,
  borderWidth: PANEL_BUTTON_BORDER_WIDTH,
};

// StagePage: a stub. StartMatchButton is deliberately a no-op — entering MatchScene isn't
// wired up yet.
export default class StagePage implements SettingsPage {
  private headerObjects: GameObj[] = [];
  private navObjects: GameObj[] = [];

  constructor(private readonly nav: SettingsPageNav) {}

  renderHeaderTitle(scene: Phaser.Scene, x: number, y: number): void {
    const title = scene.add.text(x, y, 'Stage Selection', {
      fontFamily: GAME_FONT_FAMILY,
      fontSize: `${SETTINGS_TEXT_FONT_SIZE}px`,
      color: colorToCss(0xffffff),
    });
    title.setOrigin(0, 0.5);
    this.headerObjects.push(title);
  }

  // Intentionally empty — filled in by a later spec.
  renderBody(): void {
    /* stub */
  }

  renderNav(scene: Phaser.Scene, bounds: PageBounds): void {
    destroyAll(this.navObjects);
    // Flush against the NavRegion's right edge, matching UnitPage's NextButton.
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
        0
        // No onClick — entering MatchScene from here isn't wired up yet.
      )
    );
  }

  // Back to UnitPage 2.
  handleBack(): void {
    this.nav.goBack();
  }

  destroy(): void {
    destroyAll(this.headerObjects);
    destroyAll(this.navObjects);
  }
}
