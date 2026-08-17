import Phaser from 'phaser';
import {
  ALLOWED_TILE_BOMB_ALPHA,
  ALLOWED_TILE_BOMB_COLOR,
  ALLOWED_TILE_BOMB_SELECTED_COLOR,
  ALLOWED_TILE_MOVE_ALPHA,
  ALLOWED_TILE_MOVE_COLOR,
  ALLOWED_TILE_MOVE_SELECTED_COLOR,
  DEPTH_ALLOWED_TILE_OVERLAY,
  DEPTH_TURN_COMMAND_PANEL,
  BUTTON_LABEL_MOVE,
  BUTTON_LABEL_BOMB,
  BUTTON_LABEL_BACK,
  SPRITE_BUTTON_ROW_SPACING,
  TILE_SIZE,
  GAME_CONTROL_REGION_HEIGHT,
  GAME_CONTROL_REGION_X,
  GAME_CONTROL_REGION_WIDTH,
  TURN_COMMAND_PANEL_BOTTOM_MARGIN,
} from '../constants';
import SpriteButton, { buttonFrameSize } from './spriteButton';
import { verticalButtonY } from './pillButton';
import { destroyAll } from './gameObjectUtils';
import { boardOffset } from '../rendering/boardOffset';
import type { Coordinate, TurnCmdType, TurnCommand, Unit } from '../types/api';

export interface TurnCommandPanelCallbacks {
  getAllowedTiles: (unitId: number, turnCmdType: TurnCmdType) => Promise<Coordinate[]>;
  onError: (message: string) => void;
  onConfirmedSubmit: (cmd: TurnCommand) => void;
  showConfirm: (onYes: () => void, onNo: () => void) => void;
  hideConfirm: () => void;
  isConfirmOpen: () => boolean;
  isLocked: () => boolean;
}

type ActionStackEntry =
  | { kind: 'panelOpen' }
  | { kind: 'allowedTilesShown'; turnCmdType: TurnCmdType }
  | { kind: 'confirmPending'; turnCmdType: TurnCmdType; target: Coordinate };

const BUTTON_COUNT = 3; // Move, Bomb, Back

// Total stacked height of the panel's 3 SpriteButtons; MatchSummaryPanel reads this to
// position its own button in the gap above the panel.
export function turnCommandPanelHeight(scene: Phaser.Scene): number {
  const { height } = buttonFrameSize(scene);
  return BUTTON_COUNT * height + (BUTTON_COUNT - 1) * SPRITE_BUTTON_ROW_SPACING;
}

export default class TurnCommandPanel {
  private panelObjects: SpriteButton[] = [];
  private overlayTiles: Phaser.GameObjects.Graphics[] = [];
  private actionStack: ActionStackEntry[] = [];
  private currentUnit: Unit | undefined;

  constructor(
    private readonly scene: Phaser.Scene,
    private readonly callbacks: TurnCommandPanelCallbacks
  ) {}

  openFor(unit: Unit): void {
    this.closeImmediately();
    this.currentUnit = unit;
    this.actionStack = [{ kind: 'panelOpen' }];
    this.drawPanelButtons(unit);
  }

  closeImmediately(): void {
    this.callbacks.hideConfirm();
    this.hideAllowedTiles();
    destroyAll(this.panelObjects);
    this.actionStack = [];
    this.currentUnit = undefined;
  }

  private drawPanelButtons(unit: Unit): void {
    const centerX = GAME_CONTROL_REGION_X + GAME_CONTROL_REGION_WIDTH / 2;
    const { height: buttonHeight } = buttonFrameSize(this.scene);
    const stackHeight = turnCommandPanelHeight(this.scene);
    const panelBottomY = GAME_CONTROL_REGION_HEIGHT - TURN_COMMAND_PANEL_BOTTOM_MARGIN;
    const firstCenterY = panelBottomY - stackHeight + buttonHeight / 2;

    this.drawButton(
      centerX,
      firstCenterY,
      BUTTON_LABEL_MOVE,
      !unit.hasMoved,
      buttonHeight,
      0,
      () => {
        void this.onActionButtonClick('move');
      }
    );
    this.drawButton(
      centerX,
      firstCenterY,
      BUTTON_LABEL_BOMB,
      !unit.hasUsedSkill,
      buttonHeight,
      1,
      () => {
        void this.onActionButtonClick('placeBomb');
      }
    );
    this.drawButton(centerX, firstCenterY, BUTTON_LABEL_BACK, true, buttonHeight, 2, () =>
      this.onBackButtonClick()
    );
  }

  private drawButton(
    centerX: number,
    firstCenterY: number,
    labelKey: string,
    enabled: boolean,
    buttonHeight: number,
    row: number,
    onClick: () => void
  ): void {
    const y = verticalButtonY(firstCenterY, row, buttonHeight, SPRITE_BUTTON_ROW_SPACING);
    this.panelObjects.push(
      new SpriteButton(this.scene, {
        x: centerX,
        y,
        labelKey,
        depth: DEPTH_TURN_COMMAND_PANEL,
        enabled,
        onClick,
      })
    );
  }

  // Disabled while ConfirmDialog is open — "No" is the only rollback path.
  private onBackButtonClick(): void {
    if (this.callbacks.isConfirmOpen() || this.callbacks.isLocked()) {
      return;
    }
    this.actionStack.pop();
    this.restoreTopOfStack();
  }

  private onDialogNo(): void {
    const top = this.actionStack[this.actionStack.length - 1];
    if (top?.kind === 'confirmPending') {
      this.actionStack.pop();
    }
    this.restoreTopOfStack();
  }

  private restoreTopOfStack(): void {
    this.hideAllowedTiles();
    const top = this.actionStack[this.actionStack.length - 1];
    if (!top) {
      this.closeImmediately();
      return;
    }
    if (top.kind === 'allowedTilesShown' && this.currentUnit) {
      void this.showAllowedTilesFor(this.currentUnit, top.turnCmdType, false);
    }
  }

  private async onActionButtonClick(turnCmdType: TurnCmdType): Promise<void> {
    if (this.callbacks.isConfirmOpen() || this.callbacks.isLocked() || !this.currentUnit) {
      return;
    }
    await this.showAllowedTilesFor(this.currentUnit, turnCmdType, true);
  }

  private async showAllowedTilesFor(
    unit: Unit,
    turnCmdType: TurnCmdType,
    pushStack: boolean
  ): Promise<void> {
    try {
      const tiles = await this.callbacks.getAllowedTiles(unit.id, turnCmdType);
      this.hideAllowedTiles();
      this.renderAllowedTiles(tiles, turnCmdType, unit);
      if (pushStack) {
        this.actionStack.push({ kind: 'allowedTilesShown', turnCmdType });
      }
    } catch (err) {
      this.callbacks.onError(err instanceof Error ? err.message : String(err));
    }
  }

  private renderAllowedTiles(tiles: Coordinate[], turnCmdType: TurnCmdType, unit: Unit): void {
    const fillColor = turnCmdType === 'move' ? ALLOWED_TILE_MOVE_COLOR : ALLOWED_TILE_BOMB_COLOR;
    const fillAlpha = turnCmdType === 'move' ? ALLOWED_TILE_MOVE_ALPHA : ALLOWED_TILE_BOMB_ALPHA;
    const selectedColor =
      turnCmdType === 'move' ? ALLOWED_TILE_MOVE_SELECTED_COLOR : ALLOWED_TILE_BOMB_SELECTED_COLOR;

    tiles.forEach(position => {
      const g = this.scene.add.graphics();
      g.setDepth(DEPTH_ALLOWED_TILE_OVERLAY);
      const x = position.x * TILE_SIZE + boardOffset.x;
      const y = position.y * TILE_SIZE + boardOffset.y;
      g.fillStyle(fillColor, fillAlpha);
      g.fillRect(x, y, TILE_SIZE, TILE_SIZE);

      const hitArea = new Phaser.Geom.Rectangle(x, y, TILE_SIZE, TILE_SIZE);
      g.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, px: number, py: number) =>
        Phaser.Geom.Rectangle.Contains(shape, px, py)
      );
      g.on('pointerdown', () => {
        if (this.callbacks.isConfirmOpen() || this.callbacks.isLocked()) {
          return;
        }
        g.lineStyle(1, selectedColor, 1);
        g.strokeRect(x, y, TILE_SIZE, TILE_SIZE);
        this.actionStack.push({ kind: 'confirmPending', turnCmdType, target: position });
        this.callbacks.showConfirm(
          () =>
            this.callbacks.onConfirmedSubmit({
              type: turnCmdType,
              unitId: unit.id,
              target: position,
            }),
          () => this.onDialogNo()
        );
      });

      this.overlayTiles.push(g);
    });
  }

  private hideAllowedTiles(): void {
    destroyAll(this.overlayTiles);
  }
}
