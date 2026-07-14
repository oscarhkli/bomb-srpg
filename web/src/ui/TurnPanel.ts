import type Phaser from 'phaser';
import {
  DEPTH_TURN_COMMAND_PANEL,
  GAME_FONT_FAMILY,
  TEAM_COLOR_FALLBACK,
  TEAM_COLORS,
  TURN_PANEL_HEIGHT,
  TURN_PANEL_MARGIN,
  TURN_PANEL_PADDING,
  SUDDEN_DEATH_COLOR,
  TURN_PANEL_TEXT_COLOR,
  TURN_PANEL_WIDTH,
} from '../constants';
import { destroyAll, colorToCss } from './gameObjectUtils';

const HEADER_HEIGHT = TURN_PANEL_HEIGHT / 2;
// Fixed pixel reservations for the right-aligned "NN / NN" value row so 1-vs-2-digit numbers
// don't shift layout — spec caps both turn and maxTurns at 99 (2 digits).
const VALUE_DIGIT_WIDTH = 20;
const VALUE_SLASH_WIDTH = 12;

export default class TurnPanel {
  private objects: Phaser.GameObjects.GameObject[] = [];

  constructor(private readonly scene: Phaser.Scene) {}

  update(turn: number, maxTurns: number, activeTeam: number): void {
    destroyAll(this.objects);

    const x = TURN_PANEL_MARGIN;
    const y = TURN_PANEL_MARGIN;

    const header = this.scene.add.graphics();
    header.setDepth(DEPTH_TURN_COMMAND_PANEL);
    header.setScrollFactor(0);
    header.fillStyle(TEAM_COLORS[activeTeam] ?? TEAM_COLOR_FALLBACK);
    header.fillRoundedRect(x, y, TURN_PANEL_WIDTH, HEADER_HEIGHT, 4);
    this.objects.push(header);

    const label = this.scene.add.text(x + TURN_PANEL_PADDING, y + TURN_PANEL_PADDING, 'Turn', {
      fontFamily: GAME_FONT_FAMILY,
      color: colorToCss(TURN_PANEL_TEXT_COLOR),
    });
    label.setDepth(DEPTH_TURN_COMMAND_PANEL);
    label.setScrollFactor(0);
    this.objects.push(label);

    const valueY = y + HEADER_HEIGHT;
    const rightEdge = x + TURN_PANEL_WIDTH - TURN_PANEL_PADDING;

    const maxTurnsText = this.scene.add.text(rightEdge, valueY, String(maxTurns), {
      fontFamily: GAME_FONT_FAMILY,
      color: colorToCss(TURN_PANEL_TEXT_COLOR),
    });
    maxTurnsText.setOrigin(1, 0);
    maxTurnsText.setDepth(DEPTH_TURN_COMMAND_PANEL);
    maxTurnsText.setScrollFactor(0);
    this.objects.push(maxTurnsText);

    const slashText = this.scene.add.text(rightEdge - VALUE_DIGIT_WIDTH, valueY, '/', {
      fontFamily: GAME_FONT_FAMILY,
      color: colorToCss(TURN_PANEL_TEXT_COLOR),
    });
    slashText.setOrigin(1, 0);
    slashText.setDepth(DEPTH_TURN_COMMAND_PANEL);
    slashText.setScrollFactor(0);
    this.objects.push(slashText);

    const suddenDeath = turn > maxTurns;
    const turnText = this.scene.add.text(
      rightEdge - VALUE_DIGIT_WIDTH - VALUE_SLASH_WIDTH,
      valueY,
      String(turn),
      {
        fontFamily: GAME_FONT_FAMILY,
        color: colorToCss(suddenDeath ? SUDDEN_DEATH_COLOR : TURN_PANEL_TEXT_COLOR),
      }
    );
    turnText.setOrigin(1, 0);
    turnText.setDepth(DEPTH_TURN_COMMAND_PANEL);
    turnText.setScrollFactor(0);
    this.objects.push(turnText);
  }
}
