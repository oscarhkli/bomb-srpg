import Phaser from 'phaser';
import type { MatchSettingsSceneData } from './MatchSettingsScene';
import {
  initRoom,
  initToken,
  getMatchState,
  getMatchConfig,
  getAllowedTiles,
  submitTurnCommand,
  resolveTurn,
  resetTurn,
  surrender,
  startTurn,
  rematch,
  deleteMatch,
} from '../engine/api';
import TurnCommandPanel from '../ui/TurnCommandPanel';
import ConfirmDialog from '../ui/ConfirmDialog';
import TurnPanel from '../ui/TurnPanel';
import MatchSummaryPanel from '../ui/MatchSummaryPanel';
import ErrorPanel from '../ui/ErrorPanel';
import TurnBanner from '../ui/TurnBanner';
import SuddenDeathCutscene from '../ui/SuddenDeathCutscene';
import VictoryCutscene from '../ui/VictoryCutscene';
import { playResolveTurnEvents, type BombGraphics } from '../rendering/resolveTurnPlayer';
import {
  renderTerrain,
  renderOccupants,
  renderBomb,
  tileCenter,
  type BoardRenderContext,
} from '../rendering/boardRenderer';
import {
  TILE_SIZE,
  BOMB_SIZE,
  DEPTH_OCCUPANT,
  DEPTH_SUDDEN_DEATH_BOMB,
  UNIT_MOVE_TWEEN_DURATION,
  SUDDEN_DEATH_BOMB_DROP_DURATION_MS,
  CONFIRM_TEXT_RESOLVE,
  CONFIRM_TEXT_RESET,
  CONFIRM_TEXT_SURRENDER,
  FADE_MS,
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
  // Set when re-entering via Rematch: create() calls rematch() (same room, same gameCfg)
  // before its usual getMatchState() bootstrap, and fades the scene back in.
  isRematch?: boolean;
}

export default class MatchScene extends Phaser.Scene {
  private roomId!: string;
  private playerTokens!: [string, string];
  private gameState!: GameState;
  private gameCfg!: GameCfg;
  // The terrain (grid) is painted once per scene entry into this persistent layer; tileType is
  // immutable for a match, so no operation rebuilds it. occupantObjects is the swappable layer.
  private terrainObjects: Phaser.GameObjects.GameObject[] = [];
  private occupantObjects: Phaser.GameObjects.GameObject[] = [];
  private unitGraphicsById = new Map<number, Phaser.GameObjects.Graphics>();
  private bombGraphicsById = new Map<number, BombGraphics>();
  private softBlockGraphicsById = new Map<number, Phaser.GameObjects.Graphics>();
  private allowedTilesCache = new Map<string, Coordinate[]>();
  private turnCommandPanel!: TurnCommandPanel;
  private confirmDialog!: ConfirmDialog;
  private turnPanel!: TurnPanel;
  private matchSummaryPanel!: MatchSummaryPanel;
  private errorPanel!: ErrorPanel;
  private turnBanner!: TurnBanner;
  private suddenDeathCutscene!: SuddenDeathCutscene;
  private victoryCutscene!: VictoryCutscene;
  private isSubmitting = false;
  private interactionsEnabled = false;
  // Whether MatchSummaryPanel is currently open; folded into isLocked().
  private summaryPanelOpen = false;
  // Whether a lifecycle button's (Resolve/Reset/Surrender) own ConfirmDialog is pending Yes/No.
  private lifecycleConfirmOpen = false;
  // Guards VictoryCutscene's Rematch/Return buttons against a double-click queuing two actions.
  private victoryActionTaken = false;
  // Bumped on 'shutdown'; async callbacks compare against this to detect a torn-down scene.
  private generation = 0;

  constructor() {
    super('MatchScene');
  }

  create(data: MatchSceneData): void {
    const gen = this.generation;
    this.roomId = data.roomId;
    this.playerTokens = data.playerTokens;
    this.victoryActionTaken = false;
    this.isSubmitting = false;
    this.interactionsEnabled = false;
    this.summaryPanelOpen = false;
    this.lifecycleConfirmOpen = false;
    this.events.once('shutdown', () => {
      this.generation++;
    });
    initRoom(data.roomId);
    this.confirmDialog = new ConfirmDialog(this);
    this.turnPanel = new TurnPanel(this);
    this.errorPanel = new ErrorPanel(this);
    this.turnBanner = new TurnBanner(this);
    this.suddenDeathCutscene = new SuddenDeathCutscene(this);
    this.victoryCutscene = new VictoryCutscene(this);
    this.matchSummaryPanel = new MatchSummaryPanel(this, {
      isLocked: () => this.isLocked(),
      onButtonClicked: () => this.openMatchSummaryPanel(),
      onBackButtonClicked: () => this.closeMatchSummaryPanel(),
      onResolveButtonClicked: () => this.onResolveButtonClicked(),
      onResetButtonClicked: () => this.onResetButtonClicked(),
      onSurrenderButtonClicked: () => this.onSurrenderButtonClicked(),
    });
    this.turnCommandPanel = new TurnCommandPanel(this, {
      getAllowedTiles: (unitId, turnCmdType) => this.getAllowedTilesCached(unitId, turnCmdType),
      onError: message => this.showError(message),
      onConfirmedSubmit: cmd => void this.handleTurnCommand(cmd),
      showConfirm: (onYes, onNo) => this.confirmDialog.show(onYes, onNo, 'Confirm?'),
      hideConfirm: () => this.confirmDialog.hide(),
      isConfirmOpen: () => this.confirmDialog.isOpen,
      isLocked: () => this.isLocked(),
    });

    const bootstrap = data.isRematch ? rematch().then(() => undefined) : Promise.resolve();

    bootstrap
      .then(() => getMatchState())
      .then(state => {
        if (gen !== this.generation) {
          return;
        }
        // Paint the immutable terrain layer once, then the occupants. Grid bounds are set here
        // (grid is immutable, so no later swap re-sets them).
        const { cols, rows } = renderTerrain(this.boardCtx(), state.grid);
        this.turnCommandPanel.setGridBounds(cols * TILE_SIZE, rows * TILE_SIZE);
        this.renderBoard(state);
        this.cameras.main.centerOn((cols * TILE_SIZE) / 2, (rows * TILE_SIZE) / 2);
        this.matchSummaryPanel.renderButton();
        this.refreshTurnPanelIfReady();
        void this.beginTurn();
      })
      .catch(() => {
        if (gen !== this.generation) {
          return;
        }
        this.showError('Failed to load match state');
      });

    getMatchConfig()
      .then(cfg => {
        if (gen !== this.generation) {
          return;
        }
        this.gameCfg = cfg;
        this.refreshTurnPanelIfReady();
      })
      .catch(() => {
        if (gen !== this.generation) {
          return;
        }
        this.showError('Failed to load match config');
      });

    this.cameras.main.fadeIn(FADE_MS);
  }

  // gameState and gameCfg are fetched via two independent promise chains (kept separate so
  // MatchScene's initial render doesn't wait on both round-trips), so either may resolve first.
  private refreshTurnPanelIfReady(): void {
    if (this.gameState && this.gameCfg) {
      this.turnPanel.update(this.gameState.turn, this.gameCfg.maxTurns, this.gameState.activeTeam);
    }
  }

  // Per-turn startTurn() sequence: refresh state, start the turn, play sudden-death/turn-banner
  // cutscenes. Interactions are disabled for the duration and re-enabled in `finally`.
  private async beginTurn(): Promise<void> {
    const gen = this.generation;
    this.interactionsEnabled = false;
    try {
      const state = await getMatchState();
      if (gen !== this.generation) {
        return;
      }
      this.gameState = state;
      initToken(this.playerTokens[state.activeTeam - 1]!);
      this.refreshTurnPanelIfReady();

      const resp = await startTurn();
      if (gen !== this.generation) {
        return;
      }
      if (resp.inSuddenDeath) {
        // Refetch so gameState.bombs includes the hazards startTurn() just injected server-side.
        this.gameState = await getMatchState();
        if (gen !== this.generation) {
          return;
        }
        const bombPlacedEvents = resp.gameEvents.filter(event => event.type === 'bombPlaced');
        await this.suddenDeathCutscene.play(bombPlacedEvents, event =>
          this.dropSuddenDeathBomb(event)
        );
        if (gen !== this.generation) {
          return;
        }
      }
      await this.turnBanner.play(state.activeTeam);
    } catch (err) {
      if (gen !== this.generation) {
        return;
      }
      this.showError(err instanceof Error ? err.message : String(err));
    } finally {
      if (gen === this.generation) {
        this.interactionsEnabled = true;
      }
    }
  }

  // Renders a sudden-death bomb dropping in from off-screen onto its tile, via renderBomb().
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
    // Relative to the bomb's own rest position so the drop starts fully off-screen on any tile.
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
    const gen = this.generation;
    this.isSubmitting = true;
    this.clearErrors();
    try {
      let succeeded = false;
      try {
        const gameEvents = await submitTurnCommand(cmd);
        if (gen !== this.generation) {
          return;
        }
        this.allowedTilesCache.clear();
        this.turnCommandPanel.closeImmediately();
        succeeded = true;
        for (const event of gameEvents) {
          if (!this.applyGameEvent(event)) {
            // applyGameEvent already surfaced the specific "Invalid …" error.
            succeeded = false;
            break;
          }
        }
      } catch (err) {
        if (gen !== this.generation) {
          return;
        }
        this.showError(err instanceof Error ? err.message : String(err));
        this.turnCommandPanel.closeImmediately();
      }

      await this.resyncFromServer(gen, succeeded);
      if (gen !== this.generation) {
        return;
      }
    } finally {
      if (gen === this.generation) {
        this.isSubmitting = false;
      }
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

  // Refetches gameState. On success, adopts it without redrawing (optimistic visuals already
  // match). On failure, rebuilds the occupant layer from truth to resync.
  private async resyncFromServer(
    gen: number,
    succeeded: boolean,
    afterAdopt?: (freshState: GameState) => void
  ): Promise<void> {
    try {
      const freshState = await getMatchState();
      if (gen !== this.generation) {
        return;
      }
      if (succeeded) {
        this.gameState = freshState;
      } else {
        this.renderBoard(freshState);
      }
      afterAdopt?.(freshState);
    } catch {
      if (gen !== this.generation) {
        return;
      }
      this.showError('Failed to refresh match state');
    }
  }

  // True while a server call is in flight or a fresh render hasn't finished yet.
  private isBusy(): boolean {
    return this.isSubmitting || !this.interactionsEnabled;
  }

  // Gates unit clicks, TurnCommandPanel, and MatchSummaryButton.
  private isLocked(): boolean {
    return this.isBusy() || this.summaryPanelOpen;
  }

  private openMatchSummaryPanel(): void {
    if (!this.gameCfg) {
      this.showError('Match config is still loading, please try again shortly');
      return;
    }
    this.summaryPanelOpen = true;
    this.matchSummaryPanel.open(this.gameState, this.gameCfg);
  }

  private closeMatchSummaryPanel(): void {
    this.summaryPanelOpen = false;
    this.matchSummaryPanel.close();
  }

  // Gates the Resolve/Reset/Surrender lifecycle buttons; narrower than isLocked() since these
  // are only reachable while MatchSummaryPanel is open.
  private isLifecycleButtonBusy(): boolean {
    return this.isBusy() || this.lifecycleConfirmOpen;
  }

  private showLifecycleConfirm(onYes: () => void, text: string): void {
    this.turnCommandPanel.closeImmediately();
    this.lifecycleConfirmOpen = true;
    this.confirmDialog.show(
      () => {
        this.lifecycleConfirmOpen = false;
        this.closeMatchSummaryPanel();
        onYes();
      },
      () => {
        this.lifecycleConfirmOpen = false;
        this.confirmDialog.hide();
      },
      text
    );
  }

  private onResolveButtonClicked(): void {
    if (this.isLifecycleButtonBusy()) {
      return;
    }
    this.showLifecycleConfirm(() => void this.handleResolveTurn(), CONFIRM_TEXT_RESOLVE);
  }

  private onResetButtonClicked(): void {
    if (this.isLifecycleButtonBusy()) {
      return;
    }
    this.showLifecycleConfirm(() => void this.handleResetTurn(), CONFIRM_TEXT_RESET);
  }

  private onSurrenderButtonClicked(): void {
    if (this.isLifecycleButtonBusy()) {
      return;
    }
    this.showLifecycleConfirm(() => void this.handleSurrender(), CONFIRM_TEXT_SURRENDER);
  }

  // Masks the occupant rebuild behind a camera fade-out/fade-in.
  private async handleResetTurn(): Promise<void> {
    if (this.isSubmitting) {
      return;
    }
    const gen = this.generation;
    this.isSubmitting = true;
    this.clearErrors();
    this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
    try {
      await resetTurn(); // 204 No Content — no body to adopt
      if (gen !== this.generation) {
        return;
      }
      const freshState = await getMatchState();
      if (gen !== this.generation) {
        return;
      }
      this.renderBoard(freshState); // occupant-only rebuild — terrain untouched
    } catch (err) {
      if (gen !== this.generation) {
        return;
      }
      this.showError(err instanceof Error ? err.message : String(err));
    } finally {
      if (gen === this.generation) {
        this.isSubmitting = false;
      }
      this.cameras.main.fadeIn(FADE_MS);
    }
  }

  // Reuses handleMatchEnded()'s VictoryCutscene rendering; surrender just delivers the event differently.
  private async handleSurrender(): Promise<void> {
    if (this.isSubmitting) {
      return;
    }
    const gen = this.generation;
    this.isSubmitting = true;
    this.clearErrors();
    try {
      const events = await surrender({ teamId: this.gameState.activeTeam });
      if (gen !== this.generation) {
        return;
      }
      const matchEndedEvent = events.find(event => event.type === 'matchEnded');
      if (matchEndedEvent) {
        this.handleMatchEnded(matchEndedEvent);
      } else {
        this.showError('Invalid response from surrender');
      }
    } catch (err) {
      if (gen !== this.generation) {
        return;
      }
      this.showError(err instanceof Error ? err.message : String(err));
    } finally {
      if (gen === this.generation) {
        this.isSubmitting = false;
      }
    }
  }

  private async handleResolveTurn(): Promise<void> {
    if (this.isSubmitting) {
      return;
    }
    const gen = this.generation;
    this.isSubmitting = true;
    this.clearErrors();
    let events: GameEvent[] = [];
    let succeeded = false;
    try {
      try {
        events = await resolveTurn();
        if (gen !== this.generation) {
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
          if (gen !== this.generation) {
            return;
          }
          succeeded = true;
        }
      } catch (err) {
        if (gen !== this.generation) {
          return;
        }
        this.showError(err instanceof Error ? err.message : String(err));
      }
      // turnPanel is turn-advance UI, not reconciliation, so it refreshes regardless of success.
      await this.resyncFromServer(gen, succeeded, freshState => {
        this.turnPanel.update(freshState.turn, this.gameCfg.maxTurns, freshState.activeTeam);
      });
      if (gen !== this.generation) {
        return;
      }
      const matchEndedEvent = events.find(event => event.type === 'matchEnded');
      if (matchEndedEvent) {
        this.handleMatchEnded(matchEndedEvent);
      } else {
        await this.beginTurn();
      }
    } finally {
      if (gen === this.generation) {
        this.isSubmitting = false;
      }
    }
  }

  // A missing/out-of-range winnerTeamId is a client-side integration bug: the match is over
  // server-side regardless, so there's nothing safe to resume — show the error and stop.
  private handleMatchEnded(event: GameEvent): void {
    const { winnerTeamId } = event;
    if (
      winnerTeamId === undefined ||
      (winnerTeamId !== -1 && (winnerTeamId < 1 || winnerTeamId > 2))
    ) {
      this.showError('Invalid matchEnded event received from server');
      return;
    }
    // Permanently locked: the match is over, so interactions never need to re-enable.
    this.interactionsEnabled = false;
    this.victoryCutscene.play(winnerTeamId, {
      onRematch: () => this.handleRematchClicked(),
      onReturnToSettings: () => this.handleReturnToSettingsClicked(),
    });
  }

  private handleRematchClicked(): void {
    if (this.victoryActionTaken) {
      return;
    }
    this.victoryActionTaken = true;
    this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
    this.cameras.main.once('camerafadeoutcomplete', () => {
      this.scene.restart({
        roomId: this.roomId,
        playerTokens: this.playerTokens,
        isRematch: true,
      } satisfies MatchSceneData);
    });
  }

  private handleReturnToSettingsClicked(): void {
    if (this.victoryActionTaken) {
      return;
    }
    this.victoryActionTaken = true;
    const fadeDone = new Promise<void>(resolve => {
      this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
      this.cameras.main.once('camerafadeoutcomplete', () => resolve());
    });
    const deleteDone = deleteMatch().catch(err =>
      console.error('Failed to delete match:', err instanceof Error ? err.message : err)
    );
    void Promise.all([fadeDone, deleteDone]).then(() => {
      // Forward gameCfg so MatchSettingsScene remembers the last match's settings.
      this.scene.start('MatchSettingsScene', {
        gameCfg: this.gameCfg,
      } satisfies MatchSettingsSceneData);
    });
  }

  // Destroy-and-repaint the occupant layer from server truth; terrain is never touched.
  private renderBoard(state: GameState): void {
    this.gameState = state;
    renderOccupants(this.boardCtx(), state);
  }

  // The renderer draws occupants and wires their clicks; the scene keeps ownership of state and
  // the click semantics (which is why onUnitClicked lives here, not in boardRenderer).
  private boardCtx(): BoardRenderContext {
    return {
      scene: this,
      terrainObjects: this.terrainObjects,
      occupantObjects: this.occupantObjects,
      unitGraphicsById: this.unitGraphicsById,
      bombGraphicsById: this.bombGraphicsById,
      softBlockGraphicsById: this.softBlockGraphicsById,
      onUnitClicked: unit => this.onUnitClicked(unit),
    };
  }

  // `clickedUnit` may be a stale snapshot, so resolve the live copy by id before use.
  private onUnitClicked(clickedUnit: Unit): void {
    if (this.isLocked()) {
      return;
    }
    if (this.confirmDialog.isOpen) {
      return;
    }
    const unit = this.gameState.units.find(u => u.id === clickedUnit.id) ?? clickedUnit;
    if (unit.team !== this.gameState.activeTeam) {
      console.log(`Unit ${unit.id} is clicked`, unit);
      return;
    }
    this.turnCommandPanel.openFor(unit);
  }

  private showError(message: string): void {
    this.errorPanel.show(message);
  }

  // Clears the error log at the start of each new user-initiated action.
  private clearErrors(): void {
    this.errorPanel.clear();
  }
}
