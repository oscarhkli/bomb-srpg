import Phaser from 'phaser';
import {
  initRoom,
  initToken,
  getMatchState,
  getMatchConfig,
  getAllowedTiles,
  submitTurnCommand,
  resolveTurn,
  startTurn,
} from '../engine/api';
import TurnCommandPanel from '../ui/TurnCommandPanel';
import ConfirmDialog from '../ui/ConfirmDialog';
import TurnPanel from '../ui/TurnPanel';
import ErrorPanel from '../ui/ErrorPanel';
import TurnBanner from '../ui/TurnBanner';
import SuddenDeathCutscene from '../ui/SuddenDeathCutscene';
import { drawPillButton } from '../ui/pillButton';
import { destroyAll } from '../ui/gameObjectUtils';
import { playResolveTurnEvents, type BombGraphics } from '../rendering/resolveTurnPlayer';
import {
  extractAppliedTarget,
  turnCommandTargetMatches,
  occupantsMatch,
  type AppliedTurnResult,
} from '../rendering/stateSync';
import {
  renderBoard as drawBoard,
  renderBomb,
  tileCenter,
  type BoardRenderContext,
} from '../rendering/boardRenderer';
import {
  TILE_SIZE,
  BOMB_SIZE,
  DEPTH_TURN_COMMAND_PANEL,
  DEPTH_OCCUPANT,
  DEPTH_SUDDEN_DEATH_BOMB,
  UNIT_MOVE_TWEEN_DURATION,
  SUDDEN_DEATH_BOMB_DROP_DURATION_MS,
  PANEL_BUTTON_FILL_COLOR,
  PANEL_BUTTON_FILL_ALPHA,
  PANEL_BUTTON_BORDER_COLOR,
  PANEL_BUTTON_BORDER_WIDTH,
  RESOLVE_BUTTON_WIDTH,
  RESOLVE_BUTTON_HEIGHT,
  RESOLVE_BUTTON_MARGIN_TOP,
  RESOLVE_BUTTON_LABEL,
} from '../constants';
import type {
  Coordinate,
  GameCfg,
  GameEvent,
  GameState,
  TurnCmdType,
  TurnCommand,
  Unit,
} from '../types/api';

export interface MatchSceneData {
  roomId: string;
  playerTokens: [string, string];
}

export default class MatchScene extends Phaser.Scene {
  private roomId!: string;
  private playerTokens!: [string, string];
  private gameState!: GameState;
  private gameCfg!: GameCfg;
  private boardObjects: Phaser.GameObjects.GameObject[] = [];
  private unitGraphicsById = new Map<number, Phaser.GameObjects.Graphics>();
  private bombGraphicsById = new Map<number, BombGraphics>();
  private softBlockGraphicsById = new Map<number, Phaser.GameObjects.Graphics>();
  private allowedTilesCache = new Map<string, Coordinate[]>();
  private turnCommandPanel!: TurnCommandPanel;
  private confirmDialog!: ConfirmDialog;
  private turnPanel!: TurnPanel;
  private resolveButtonObjects: Phaser.GameObjects.GameObject[] = [];
  private errorPanel!: ErrorPanel;
  private turnBanner!: TurnBanner;
  private suddenDeathCutscene!: SuddenDeathCutscene;
  private isSubmitting = false;
  private interactionsEnabled = false;
  private destroyed = false;

  constructor() {
    super('MatchScene');
  }

  create(data: MatchSceneData): void {
    this.roomId = data.roomId;
    this.playerTokens = data.playerTokens;
    this.events.once('shutdown', () => {
      this.destroyed = true;
    });
    initRoom(data.roomId);
    this.confirmDialog = new ConfirmDialog(this);
    this.turnPanel = new TurnPanel(this);
    this.errorPanel = new ErrorPanel(this);
    this.turnBanner = new TurnBanner(this);
    this.suddenDeathCutscene = new SuddenDeathCutscene(this);
    this.turnCommandPanel = new TurnCommandPanel(this, {
      getAllowedTiles: (unitId, turnCmdType) => this.getAllowedTilesCached(unitId, turnCmdType),
      onError: message => this.showError(message),
      onConfirmedSubmit: cmd => void this.handleTurnCommand(cmd),
      showConfirm: (onYes, onNo) => this.confirmDialog.show(onYes, onNo, 'Confirm?'),
      hideConfirm: () => this.confirmDialog.hide(),
      isConfirmOpen: () => this.confirmDialog.isOpen,
    });

    getMatchState()
      .then(state => {
        const { cols, rows } = this.renderBoard(state);
        this.cameras.main.centerOn((cols * TILE_SIZE) / 2, (rows * TILE_SIZE) / 2);
        this.renderResolveButton();
        this.refreshTurnPanelIfReady();
        void this.beginTurn();
      })
      .catch(() => {
        this.showError('Failed to load match state');
      });

    getMatchConfig()
      .then(cfg => {
        this.gameCfg = cfg;
        this.refreshTurnPanelIfReady();
      })
      .catch(() => {
        this.showError('Failed to load match config');
      });
  }

  // gameState and gameCfg are fetched via two independent promise chains (kept separate so
  // MatchScene's initial render doesn't wait on both round-trips), so either may resolve first.
  private refreshTurnPanelIfReady(): void {
    if (this.gameState && this.gameCfg) {
      this.turnPanel.update(this.gameState.turn, this.gameCfg.maxTurns, this.gameState.activeTeam);
    }
  }

  // Per-turn startTurn() sequence (spec005 Game Loop 5.1-5.4): refresh state, init the active
  // team's token, call startTurn(), play the sudden-death cutscene (if triggered) then the turn
  // banner, all sequentially. All interactions are disabled for the duration so no click can
  // race a turn transition (AC#5); re-enabled in `finally` so a failed startTurn() never
  // deadlocks the scene.
  private async beginTurn(): Promise<void> {
    this.interactionsEnabled = false;
    try {
      const state = await getMatchState();
      if (this.destroyed) {
        return;
      }
      this.gameState = state;
      initToken(this.playerTokens[state.activeTeam - 1]!);
      this.refreshTurnPanelIfReady();

      const resp = await startTurn();
      if (this.destroyed) {
        return;
      }
      if (resp.inSuddenDeath) {
        // injectSuddenDeathHazards() has already committed the new bombs server-side by the
        // time startTurn() resolves, so refetching now keeps gameState.bombs in sync with what
        // dropSuddenDeathBomb() is about to render — otherwise a later resolveTurn() would
        // reference a bombId this.gameState doesn't know about (spec005 gap).
        this.gameState = await getMatchState();
        if (this.destroyed) {
          return;
        }
        const bombPlacedEvents = resp.gameEvents.filter(event => event.type === 'bombPlaced');
        await this.suddenDeathCutscene.play(bombPlacedEvents, event =>
          this.dropSuddenDeathBomb(event)
        );
        if (this.destroyed) {
          return;
        }
      }
      await this.turnBanner.play(state.activeTeam);
    } catch (err) {
      if (this.destroyed) {
        return;
      }
      this.showError(err instanceof Error ? err.message : String(err));
    } finally {
      this.interactionsEnabled = true;
    }
  }

  // Renders a sudden-death-injected bomb dropping in from off-screen, resting on its tile once
  // the drop tween completes. Reuses renderBomb() so the bomb is registered in bombGraphicsById
  // exactly like a normally-placed bomb.
  private dropSuddenDeathBomb(event: GameEvent): Promise<void> {
    const { unitId, bombId, position, countdown } = event;
    if (bombId === undefined || !position || countdown === undefined || !this.inBounds(position)) {
      this.showError('Invalid bombPlaced event received from server');
      return Promise.resolve();
    }

    renderBomb(this.boardCtx(), {
      id: bombId,
      ownerId: unitId ?? 0,
      position,
      range: event.range ?? 0,
      countdown,
    });

    const bomb = this.bombGraphicsById.get(bombId);
    if (!bomb) {
      this.showError('Invalid bombPlaced event received from server');
      return Promise.resolve();
    }

    const restY = bomb.container.y;
    // Offset relative to the bomb's own rest position (not a fixed camera-height offset) so the
    // drop always starts fully off-screen with a BOMB_SIZE margin, regardless of which tile the
    // bomb lands on.
    const dropOffset = restY + BOMB_SIZE;
    bomb.container.y -= dropOffset;
    bomb.container.setDepth(DEPTH_SUDDEN_DEATH_BOMB);

    return new Promise(resolve => {
      this.tweens.add({
        targets: bomb.container,
        y: restY,
        duration: SUDDEN_DEATH_BOMB_DROP_DURATION_MS,
        ease: 'Linear',
        onComplete: () => {
          bomb.container.setDepth(DEPTH_OCCUPANT);
          resolve();
        },
      });
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

  private async handleTurnCommand(cmd: TurnCommand): Promise<void> {
    if (this.isSubmitting) {
      return;
    }
    this.isSubmitting = true;
    this.clearErrors();
    try {
      let applied: AppliedTurnResult | undefined;
      try {
        const gameEvents = await submitTurnCommand(cmd);
        this.allowedTilesCache.clear();
        this.turnCommandPanel.closeImmediately();
        for (const event of gameEvents) {
          const ok = this.applyGameEvent(event);
          if (!ok) {
            break;
          }
        }
        applied = extractAppliedTarget(cmd, gameEvents);
      } catch (err) {
        this.showError(err instanceof Error ? err.message : String(err));
        this.turnCommandPanel.closeImmediately();
      }

      await this.refreshFinalSanityCheck(applied);
    } finally {
      this.isSubmitting = false;
    }
  }

  private inBounds(c: Coordinate): boolean {
    return this.gameState.grid[c.y]?.[c.x] !== undefined;
  }

  private applyGameEvent(event: GameEvent): boolean {
    switch (event.type) {
      case 'unitMoved':
        return this.applyUnitMoved(event);
      case 'bombPlaced':
        return this.applyBombPlaced(event);
      default:
        return true;
    }
  }

  private applyUnitMoved(event: GameEvent): boolean {
    const { unitId, from, to } = event;

    if (unitId === undefined || !from || !to || !this.inBounds(from) || !this.inBounds(to)) {
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

    return true;
  }

  private applyBombPlaced(event: GameEvent): boolean {
    const { unitId, bombId, position, countdown, range } = event;

    if (
      unitId === undefined ||
      bombId === undefined ||
      !position ||
      countdown === undefined ||
      !this.inBounds(position)
    ) {
      this.showError('Invalid bombPlaced event received from server');
      return false;
    }

    renderBomb(this.boardCtx(), {
      id: bombId,
      ownerId: unitId,
      position,
      range: range ?? 0,
      countdown,
    });

    return true;
  }

  private async refreshFinalSanityCheck(applied: AppliedTurnResult | undefined): Promise<void> {
    try {
      const freshState = await getMatchState();
      if (!applied || !turnCommandTargetMatches(freshState, applied)) {
        this.showError('Match state is out of sync with the server');
        this.renderBoard(freshState);
      } else {
        this.gameState = freshState;
      }
    } catch {
      this.showError('Failed to refresh match state');
    }
  }

  private renderResolveButton(): void {
    destroyAll(this.resolveButtonObjects);
    const { width } = this.cameras.main;
    const x = width / 2 - RESOLVE_BUTTON_WIDTH / 2;
    this.resolveButtonObjects = drawPillButton(
      this,
      x,
      RESOLVE_BUTTON_MARGIN_TOP,
      RESOLVE_BUTTON_WIDTH,
      RESOLVE_BUTTON_HEIGHT,
      RESOLVE_BUTTON_LABEL,
      {
        fillColor: PANEL_BUTTON_FILL_COLOR,
        fillAlpha: PANEL_BUTTON_FILL_ALPHA,
        borderColor: PANEL_BUTTON_BORDER_COLOR,
        borderWidth: PANEL_BUTTON_BORDER_WIDTH,
      },
      DEPTH_TURN_COMMAND_PANEL,
      () => this.onResolveButtonClicked(),
      0
    );
  }

  private onResolveButtonClicked(): void {
    if (!this.interactionsEnabled) {
      return;
    }
    if (this.isSubmitting) {
      return;
    }
    if (!this.gameCfg) {
      this.showError('Match config is still loading, please try again shortly');
      return;
    }
    this.turnCommandPanel.closeImmediately();
    this.confirmDialog.show(
      () => void this.handleResolveTurn(),
      () => this.confirmDialog.hide(),
      'Confirm to end this turn?'
    );
  }

  private async handleResolveTurn(): Promise<void> {
    if (this.isSubmitting) {
      return;
    }
    this.isSubmitting = true;
    this.clearErrors();
    try {
      try {
        const events = await resolveTurn();
        if (this.destroyed) {
          return;
        }
        const { ok, done } = playResolveTurnEvents(events, {
          scene: this,
          gameStateSnapshot: this.gameState,
          unitGraphicsById: this.unitGraphicsById,
          bombGraphicsById: this.bombGraphicsById,
          softBlockGraphicsById: this.softBlockGraphicsById,
          onError: message => this.showError(message),
        });
        if (ok) {
          await done;
          if (this.destroyed) {
            return;
          }
        }
      } catch (err) {
        if (this.destroyed) {
          return;
        }
        this.showError(err instanceof Error ? err.message : String(err));
      }
      await this.refreshFinalSanityCheckAfterResolve();
      if (this.destroyed) {
        return;
      }
      // Victory-check gating is spec006; for now every resolved turn loops back into the
      // next turn's startTurn() sequence.
      await this.beginTurn();
    } finally {
      this.isSubmitting = false;
    }
  }

  private async refreshFinalSanityCheckAfterResolve(): Promise<void> {
    try {
      const freshState = await getMatchState();
      if (
        !occupantsMatch(
          freshState,
          this.unitGraphicsById,
          this.bombGraphicsById,
          this.softBlockGraphicsById
        )
      ) {
        this.showError('Match state is out of sync with the server');
      }
      this.renderBoard(freshState);
      this.turnPanel.update(freshState.turn, this.gameCfg.maxTurns, freshState.activeTeam);
      this.renderResolveButton();
    } catch {
      this.showError('Failed to refresh match state');
    }
  }

  // Redraws the whole board and keeps the scene's grid-dependent state in sync. Returns the
  // grid dimensions so create() can center the camera from the same source.
  private renderBoard(state: GameState): { cols: number; rows: number } {
    this.gameState = state;
    const dims = drawBoard(this.boardCtx(), state);
    this.turnCommandPanel.setGridBounds(dims.cols * TILE_SIZE, dims.rows * TILE_SIZE);
    return dims;
  }

  // The renderer draws occupants and wires their clicks; the scene keeps ownership of state and
  // the click semantics (which is why onUnitClicked lives here, not in boardRenderer).
  private boardCtx(): BoardRenderContext {
    return {
      scene: this,
      boardObjects: this.boardObjects,
      unitGraphicsById: this.unitGraphicsById,
      bombGraphicsById: this.bombGraphicsById,
      softBlockGraphicsById: this.softBlockGraphicsById,
      onUnitClicked: unit => this.onUnitClicked(unit),
    };
  }

  private onUnitClicked(unit: Unit): void {
    if (!this.interactionsEnabled) {
      return;
    }
    if (this.confirmDialog.isOpen) {
      return;
    }
    if (unit.team !== this.gameState.activeTeam) {
      console.log(`Unit ${unit.id} is clicked`, unit);
      return;
    }
    this.turnCommandPanel.openFor(unit);
  }

  private showError(message: string): void {
    this.errorPanel.show(message);
  }

  // Clears the error log at the start of each new user-initiated action (not inside
  // renderBoard(), since some flows call showError() immediately before a synchronous
  // renderBoard() — clearing there would destroy the message before it's ever seen).
  private clearErrors(): void {
    this.errorPanel.clear();
  }
}
