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
  DISABLED_BUTTON_COLOR,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_HEIGHT,
  PANEL_BUTTON_SPACING,
  PANEL_BUTTON_WIDTH,
  TILE_SIZE,
  TURN_COMMAND_PANEL_GUTTER,
  TURN_COMMAND_PANEL_HEIGHT,
} from '../constants';
import { drawPillButton } from './pillButton';
import type { Coordinate, TurnCmdType, Unit } from '../types/api';

export interface TurnCommandPanelCallbacks {
  getAllowedTiles: (unitId: number, turnCmdType: TurnCmdType) => Promise<Coordinate[]>;
  onError: (message: string) => void;
  onConfirmedSubmit: (turnCmdType: TurnCmdType, unitId: number, target: Coordinate) => void;
  showConfirm: (onYes: () => void, onNo: () => void) => void;
  hideConfirm: () => void;
  isConfirmOpen: () => boolean;
}

type ActionStackEntry =
  | { kind: 'panelOpen' }
  | { kind: 'allowedTilesShown'; turnCmdType: TurnCmdType }
  | { kind: 'confirmPending'; turnCmdType: TurnCmdType; target: Coordinate };

export default class TurnCommandPanel {
  private panelObjects: Phaser.GameObjects.GameObject[] = [];
  private overlayTiles: Phaser.GameObjects.Graphics[] = [];
  private actionStack: ActionStackEntry[] = [];
  private currentUnit: Unit | undefined;
  private gridWidthPx = 0;
  private gridHeightPx = 0;

  constructor(
    private readonly scene: Phaser.Scene,
    private readonly callbacks: TurnCommandPanelCallbacks
  ) {}

  setGridBounds(gridWidthPx: number, gridHeightPx: number): void {
    this.gridWidthPx = gridWidthPx;
    this.gridHeightPx = gridHeightPx;
  }

  openFor(unit: Unit): void {
    this.closeImmediately();
    this.currentUnit = unit;
    this.actionStack = [{ kind: 'panelOpen' }];
    this.drawPanelButtons(unit);
  }

  closeImmediately(): void {
    this.callbacks.hideConfirm();
    this.hideAllowedTiles();
    this.panelObjects.forEach(obj => obj.destroy());
    this.panelObjects = [];
    this.actionStack = [];
    this.currentUnit = undefined;
  }

  private drawPanelButtons(unit: Unit): void {
    const panelX = this.gridWidthPx + TURN_COMMAND_PANEL_GUTTER;
    const panelY = this.gridHeightPx - TURN_COMMAND_PANEL_HEIGHT;

    this.drawButton(panelX, panelY, 'Move', !unit.hasMoved, () => {
      void this.onActionButtonClick('move');
    });
    this.drawButton(
      panelX + PANEL_BUTTON_WIDTH + PANEL_BUTTON_SPACING,
      panelY,
      'Bomb',
      !unit.hasUsedSkill,
      () => {
        void this.onActionButtonClick('placeBomb');
      }
    );
    this.drawButton(
      panelX + PANEL_BUTTON_WIDTH + PANEL_BUTTON_SPACING,
      panelY + PANEL_BUTTON_HEIGHT + PANEL_BUTTON_SPACING,
      'Back',
      true,
      () => this.onBackButtonClick()
    );
  }

  private drawButton(
    x: number,
    y: number,
    label: string,
    enabled: boolean,
    onClick: () => void
  ): void {
    const style = {
      fillColor: enabled ? PANEL_BUTTON_FILL_COLOR : DISABLED_BUTTON_COLOR,
      fillAlpha: PANEL_BUTTON_FILL_ALPHA,
      borderColor: enabled ? PANEL_BUTTON_BORDER_COLOR : DISABLED_BUTTON_COLOR,
      borderWidth: PANEL_BUTTON_BORDER_WIDTH,
    };
    const objects = drawPillButton(
      this.scene,
      x,
      y,
      PANEL_BUTTON_WIDTH,
      PANEL_BUTTON_HEIGHT,
      label,
      style,
      DEPTH_TURN_COMMAND_PANEL,
      enabled ? onClick : undefined
    );
    this.panelObjects.push(...objects);
  }

  // Per spec: while ConfirmDialog is open, the panel's own buttons (including Back) are not
  // interactive — "No" is the only rollback path out of the confirmPending state.
  private onBackButtonClick(): void {
    if (this.callbacks.isConfirmOpen()) {
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
    if (this.callbacks.isConfirmOpen() || !this.currentUnit) {
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
      const x = position.x * TILE_SIZE;
      const y = position.y * TILE_SIZE;
      g.fillStyle(fillColor, fillAlpha);
      g.fillRect(x, y, TILE_SIZE, TILE_SIZE);

      const hitArea = new Phaser.Geom.Rectangle(x, y, TILE_SIZE, TILE_SIZE);
      g.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, px: number, py: number) =>
        Phaser.Geom.Rectangle.Contains(shape, px, py)
      );
      g.on('pointerdown', () => {
        if (this.callbacks.isConfirmOpen()) {
          return;
        }
        g.lineStyle(1, selectedColor, 1);
        g.strokeRect(x, y, TILE_SIZE, TILE_SIZE);
        this.actionStack.push({ kind: 'confirmPending', turnCmdType, target: position });
        this.callbacks.showConfirm(
          () => this.callbacks.onConfirmedSubmit(turnCmdType, unit.id, position),
          () => this.onDialogNo()
        );
      });

      this.overlayTiles.push(g);
    });
  }

  private hideAllowedTiles(): void {
    this.overlayTiles.forEach(g => g.destroy());
    this.overlayTiles = [];
  }
}
