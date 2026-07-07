import Phaser from 'phaser';
import {
  initRoom,
  initToken,
  getMatchState,
  getAllowedTiles,
  submitTurnCommand,
} from '../engine/api';
import TurnCommandPanel from '../ui/TurnCommandPanel';
import ConfirmDialog from '../ui/ConfirmDialog';
import {
  TILE_SIZE,
  TERRAIN_COLORS,
  TERRAIN_BORDER_COLOR,
  TEAM_COLORS,
  UNIT_SIZE,
  OCCUPANT_STROKE_COLOR,
  OCCUPANT_ICON_RADIUS,
  OCCUPANT_ICON_STROKE_WIDTH,
  SOFTBLOCK_COLOR,
  SOFTBLOCK_SIZE,
  SOFTBLOCK_CORNER_RADIUS,
  BOMB_COLOR,
  BOMB_SIZE,
  DEPTH_GRID,
  DEPTH_OCCUPANT,
  UNIT_MOVE_TWEEN_DURATION,
} from '../constants';
import type {
  Coordinate,
  GameEvent,
  GameState,
  Tile,
  TurnCmdType,
  TurnCommand,
  Unit,
  SoftBlock,
  Bomb,
} from '../types/api';

export interface MatchSceneData {
  roomId: string;
  playerTokens: [string, string];
}

export default class MatchScene extends Phaser.Scene {
  private roomId!: string;
  private playerTokens!: [string, string];
  private gameState!: GameState;
  private boardObjects: Phaser.GameObjects.GameObject[] = [];
  private unitGraphicsById = new Map<number, Phaser.GameObjects.Graphics>();
  private allowedTilesCache = new Map<string, Coordinate[]>();
  private turnCommandPanel!: TurnCommandPanel;
  private confirmDialog!: ConfirmDialog;
  private isSubmitting = false;

  constructor() {
    super('MatchScene');
  }

  create(data: MatchSceneData): void {
    this.roomId = data.roomId;
    this.playerTokens = data.playerTokens;
    console.log('roomId:', this.roomId, 'playerTokens:', this.playerTokens);
    initRoom(data.roomId);
    this.confirmDialog = new ConfirmDialog(this);
    this.turnCommandPanel = new TurnCommandPanel(this, {
      getAllowedTiles: (unitId, turnCmdType) => this.getAllowedTilesCached(unitId, turnCmdType),
      onError: message => this.showError(message),
      onConfirmedSubmit: (turnCmdType, unitId, target) =>
        this.onConfirmedSubmit(turnCmdType, unitId, target),
      showConfirm: (onYes, onNo) => this.confirmDialog.show(onYes, onNo),
      hideConfirm: () => this.confirmDialog.hide(),
      isConfirmOpen: () => this.confirmDialog.isOpen,
    });

    getMatchState()
      .then(state => {
        this.renderBoard(state);
        this.centerCamera(state.grid);
      })
      .catch(() => {
        this.showError('Failed to load match state');
      });
  }

  private async getAllowedTilesCached(
    unitId: number,
    turnCmdType: TurnCmdType
  ): Promise<Coordinate[]> {
    const key = `${unitId}:${turnCmdType}`;
    const cached = this.allowedTilesCache.get(key);
    if (cached) {
      return cached;
    }
    const tiles = await getAllowedTiles({ unitId, turnCmdType });
    this.allowedTilesCache.set(key, tiles);
    return tiles;
  }

  private onConfirmedSubmit(turnCmdType: TurnCmdType, unitId: number, target: Coordinate): void {
    void this.handleTurnCommand({ type: turnCmdType, unitId, target });
  }

  private async handleTurnCommand(cmd: TurnCommand): Promise<void> {
    if (this.isSubmitting) {
      return;
    }
    this.isSubmitting = true;
    try {
      const predictedGrid = cloneGrid(this.gameState.grid);

      try {
        const gameEvents = await submitTurnCommand(cmd);
        this.allowedTilesCache.clear();
        this.turnCommandPanel.closeImmediately();
        for (const event of gameEvents) {
          const ok = this.applyGameEvent(event, cmd.unitId, predictedGrid);
          if (!ok) {
            break;
          }
        }
      } catch (err) {
        this.showError(err instanceof Error ? err.message : String(err));
        this.turnCommandPanel.closeImmediately();
      }

      await this.refreshFinalSanityCheck(predictedGrid);
    } finally {
      this.isSubmitting = false;
    }
  }

  private applyGameEvent(event: GameEvent, actorUnitId: number, predictedGrid: Tile[][]): boolean {
    switch (event.type) {
      case 'unitMoved':
        return this.applyUnitMoved(event, actorUnitId, predictedGrid);
      case 'bombPlaced':
        return this.applyBombPlaced(event, actorUnitId, predictedGrid);
      default:
        return true;
    }
  }

  private applyUnitMoved(event: GameEvent, actorUnitId: number, predictedGrid: Tile[][]): boolean {
    const { unitId, from, to } = event;
    const fromTile = from && this.gameState.grid[from.y]?.[from.x];
    const toTile = to && this.gameState.grid[to.y]?.[to.x];
    const actorAtFrom =
      from &&
      this.gameState.units.some(
        u => u.id === unitId && u.position.x === from.x && u.position.y === from.y
      );

    if (
      unitId === undefined ||
      !from ||
      !to ||
      unitId !== actorUnitId ||
      fromTile?.occupantType !== 'OccupantUnit' ||
      fromTile.occupantId !== unitId ||
      toTile?.occupantType !== 'OccupantNone' ||
      !actorAtFrom
    ) {
      this.showError('Invalid unitMoved event received from server');
      return false;
    }

    const g = this.unitGraphicsById.get(unitId);
    if (g) {
      const fromCenter = tileCenter(from);
      const toCenter = tileCenter(to);
      this.tweens.add({
        targets: g,
        x: g.x + (toCenter.cx - fromCenter.cx),
        y: g.y + (toCenter.cy - fromCenter.cy),
        duration: UNIT_MOVE_TWEEN_DURATION,
        ease: 'Linear',
      });
    }

    const predictedFrom = predictedGrid[from.y]?.[from.x];
    const predictedTo = predictedGrid[to.y]?.[to.x];
    if (predictedFrom) {
      predictedFrom.occupantType = 'OccupantNone';
      predictedFrom.occupantId = 0;
    }
    if (predictedTo) {
      predictedTo.occupantType = 'OccupantUnit';
      predictedTo.occupantId = unitId;
    }

    return true;
  }

  private applyBombPlaced(event: GameEvent, actorUnitId: number, predictedGrid: Tile[][]): boolean {
    const { unitId, bombId, position, countdown, range } = event;
    const targetTile = position && this.gameState.grid[position.y]?.[position.x];

    if (
      unitId === undefined ||
      bombId === undefined ||
      !position ||
      countdown === undefined ||
      unitId !== actorUnitId ||
      targetTile?.occupantType !== 'OccupantNone'
    ) {
      this.showError('Invalid bombPlaced event received from server');
      return false;
    }

    this.renderBomb({ id: bombId, ownerId: unitId, position, range: range ?? 0, countdown });

    const predictedTile = predictedGrid[position.y]?.[position.x];
    if (predictedTile) {
      predictedTile.occupantType = 'OccupantBomb';
      predictedTile.occupantId = bombId;
    }

    return true;
  }

  private async refreshFinalSanityCheck(predictedGrid: Tile[][]): Promise<void> {
    try {
      const freshState = await getMatchState();
      if (!gridsEqual(freshState.grid, predictedGrid)) {
        this.showError('Match state is out of sync with the server');
        this.renderBoard(freshState);
      } else {
        this.gameState = freshState;
      }
    } catch {
      this.showError('Failed to refresh match state');
    }
  }

  private renderBoard(state: GameState): void {
    this.boardObjects.forEach(obj => obj.destroy());
    this.boardObjects = [];
    this.unitGraphicsById.clear();
    this.gameState = state;

    this.renderGrid(state.grid);
    this.renderUnits(state.units);
    this.renderSoftBlocks(state.softBlocks);
    this.renderBombs(state.bombs);

    const cols = state.grid[0]?.length ?? 0;
    const rows = state.grid.length;
    this.turnCommandPanel.setGridBounds(cols * TILE_SIZE, rows * TILE_SIZE);
  }

  private renderGrid(grid: Tile[][]): void {
    const g = this.add.graphics();
    g.setDepth(DEPTH_GRID);
    this.boardObjects.push(g);
    g.lineStyle(1, TERRAIN_BORDER_COLOR);
    for (let row = 0; row < grid.length; row++) {
      const rowTiles = grid[row];
      if (!rowTiles) {
        continue;
      }
      for (let col = 0; col < rowTiles.length; col++) {
        const tile = rowTiles[col];
        if (!tile) {
          continue;
        }
        const x = col * TILE_SIZE;
        const y = row * TILE_SIZE;
        g.fillStyle(TERRAIN_COLORS[tile.type]);
        g.fillRect(x, y, TILE_SIZE, TILE_SIZE);
        g.strokeRect(x, y, TILE_SIZE, TILE_SIZE);
      }
    }
  }

  private renderUnits(units: Unit[]): void {
    units
      .filter(unit => unit.hp > 0)
      .forEach(unit => {
        const { cx, cy } = tileCenter(unit.position);
        const g = this.add.graphics();
        g.setDepth(DEPTH_OCCUPANT);
        this.boardObjects.push(g);
        this.unitGraphicsById.set(unit.id, g);
        const teamColor = TEAM_COLORS[unit.team];
        if (teamColor === undefined) {
          console.warn(`Unit ${unit.id} has unconfigured team ${unit.team}, rendering as white`);
        }
        g.fillStyle(teamColor ?? 0xffffff);
        g.fillRect(cx - UNIT_SIZE / 2, cy - UNIT_SIZE / 2, UNIT_SIZE, UNIT_SIZE);
        this.drawArchetypeIcon(g, unit.type, cx, cy);
        this.attachUnitClickHandler(g, unit);
      });
  }

  private attachUnitClickHandler(g: Phaser.GameObjects.Graphics, unit: Unit): void {
    const hitArea = new Phaser.Geom.Rectangle(
      unit.position.x * TILE_SIZE,
      unit.position.y * TILE_SIZE,
      TILE_SIZE,
      TILE_SIZE
    );
    g.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, x: number, y: number) =>
      Phaser.Geom.Rectangle.Contains(shape, x, y)
    );
    g.on('pointerdown', () => this.onUnitClicked(unit));
  }

  private onUnitClicked(unit: Unit): void {
    if (this.confirmDialog.isOpen) {
      return;
    }
    if (unit.team !== this.gameState.activeTeam) {
      console.log(`Unit ${unit.id} is clicked`, unit);
      return;
    }
    initToken(this.playerTokens[this.gameState.activeTeam - 1]!);
    this.turnCommandPanel.openFor(unit);
  }

  private renderSoftBlocks(softBlocks: SoftBlock[]): void {
    softBlocks.forEach(block => {
      const { cx, cy } = tileCenter(block.position);
      const g = this.add.graphics();
      g.setDepth(DEPTH_OCCUPANT);
      this.boardObjects.push(g);
      g.fillStyle(SOFTBLOCK_COLOR);
      g.fillRoundedRect(
        cx - SOFTBLOCK_SIZE / 2,
        cy - SOFTBLOCK_SIZE / 2,
        SOFTBLOCK_SIZE,
        SOFTBLOCK_SIZE,
        SOFTBLOCK_CORNER_RADIUS
      );
      this.attachClickLogger(g, block.position, `SoftBlock ${block.id}`, block);
    });
  }

  private renderBombs(bombs: Bomb[]): void {
    bombs.forEach(bomb => this.renderBomb(bomb));
  }

  private renderBomb(bomb: Bomb): void {
    const { cx, cy } = tileCenter(bomb.position);
    const g = this.add.graphics();
    g.setDepth(DEPTH_OCCUPANT);
    this.boardObjects.push(g);
    g.fillStyle(BOMB_COLOR);
    g.fillCircle(cx, cy, BOMB_SIZE / 2);
    const text = this.add.text(cx, cy, String(bomb.countdown), { color: '#ffffff' });
    text.setOrigin(0.5);
    text.setDepth(DEPTH_OCCUPANT);
    this.boardObjects.push(text);
    this.attachClickLogger(g, bomb.position, `Bomb ${bomb.id}`, bomb);
  }

  private attachClickLogger(
    g: Phaser.GameObjects.Graphics,
    position: Coordinate,
    label: string,
    details: unknown
  ): void {
    const hitArea = new Phaser.Geom.Rectangle(
      position.x * TILE_SIZE,
      position.y * TILE_SIZE,
      TILE_SIZE,
      TILE_SIZE
    );
    g.setInteractive(hitArea, (shape: Phaser.Geom.Rectangle, x: number, y: number) =>
      Phaser.Geom.Rectangle.Contains(shape, x, y)
    );
    g.on('pointerdown', () => console.log(`${label} is clicked`, details));
  }

  private drawArchetypeIcon(
    g: Phaser.GameObjects.Graphics,
    archetype: string,
    cx: number,
    cy: number
  ): void {
    g.lineStyle(OCCUPANT_ICON_STROKE_WIDTH, OCCUPANT_STROKE_COLOR);
    switch (archetype) {
      case 'Bandit':
        g.strokeCircle(cx, cy, OCCUPANT_ICON_RADIUS);
        break;
      case 'Witch':
        g.strokePoints(regularPolygonPoints(cx, cy, 3, OCCUPANT_ICON_RADIUS), true);
        break;
      case 'Fighter':
        g.strokePoints(regularPolygonPoints(cx, cy, 5, OCCUPANT_ICON_RADIUS), true);
        break;
      case 'King':
        g.strokePoints(starPoints(cx, cy, 5, OCCUPANT_ICON_RADIUS, 4), true);
        break;
      default:
        console.warn(`Unrecognized archetype "${archetype}", drawing no icon`);
    }
  }

  private centerCamera(grid: Tile[][]): void {
    const cols = grid[0]?.length ?? 0;
    const rows = grid.length;
    this.cameras.main.centerOn((cols * TILE_SIZE) / 2, (rows * TILE_SIZE) / 2);
  }

  private showError(message: string): void {
    const { width, height } = this.cameras.main;
    this.add.text(width / 2, height / 2, message).setOrigin(0.5);
  }
}

function cloneGrid(grid: Tile[][]): Tile[][] {
  return grid.map(row => row.map(tile => ({ ...tile })));
}

function gridsEqual(a: Tile[][], b: Tile[][]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  for (let y = 0; y < a.length; y++) {
    const rowA = a[y];
    const rowB = b[y];
    if (!rowA || rowA.length !== rowB?.length) {
      return false;
    }
    for (let x = 0; x < rowA.length; x++) {
      const tileA = rowA[x];
      const tileB = rowB[x];
      if (
        !tileA ||
        tileA.type !== tileB?.type ||
        tileA.occupantType !== tileB.occupantType ||
        tileA.occupantId !== tileB.occupantId
      ) {
        return false;
      }
    }
  }
  return true;
}

function tileCenter(position: Coordinate): { cx: number; cy: number } {
  return {
    cx: position.x * TILE_SIZE + TILE_SIZE / 2,
    cy: position.y * TILE_SIZE + TILE_SIZE / 2,
  };
}

// Vertices of a regular polygon centered at (cx, cy), first vertex pointing straight up.
function regularPolygonPoints(
  cx: number,
  cy: number,
  sides: number,
  radius: number
): Phaser.Math.Vector2[] {
  return Array.from({ length: sides }, (_, i) => {
    const angle = -Math.PI / 2 + (i * 2 * Math.PI) / sides;
    return new Phaser.Math.Vector2(cx + radius * Math.cos(angle), cy + radius * Math.sin(angle));
  });
}

// Vertices of a 5-pointed star centered at (cx, cy), alternating outer/inner radius,
// first vertex pointing straight up.
function starPoints(
  cx: number,
  cy: number,
  points: number,
  outerRadius: number,
  innerRadius: number
): Phaser.Math.Vector2[] {
  const vertices: Phaser.Math.Vector2[] = [];
  for (let i = 0; i < points * 2; i++) {
    const radius = i % 2 === 0 ? outerRadius : innerRadius;
    const angle = -Math.PI / 2 + (i * Math.PI) / points;
    vertices.push(
      new Phaser.Math.Vector2(cx + radius * Math.cos(angle), cy + radius * Math.sin(angle))
    );
  }
  return vertices;
}
