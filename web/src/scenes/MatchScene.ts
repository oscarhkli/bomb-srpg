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
  consumeCpuStatus,
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
import { SPRITE_MANIFEST, SPRITE_ASSET_BASE } from '../rendering/spriteManifest';
import {
  establishBoardLayout,
  renderStageBackground,
  renderOccupants,
  renderBomb,
  tileCenter,
  occupantDepth,
  type BoardRenderContext,
} from '../rendering/boardRenderer';
import {
  BOMB_SIZE,
  DEPTH_SUDDEN_DEATH_BOMB,
  UNIT_MOVE_TWEEN_DURATION,
  SUDDEN_DEATH_BOMB_DROP_DURATION_MS,
  CONFIRM_TEXT_RESOLVE,
  CONFIRM_TEXT_RESET,
  CONFIRM_TEXT_SURRENDER,
  FADE_MS,
  CPU_POLL_BACKOFF_MS,
  CPU_POLL_BUDGET_MS,
  CPU_PLAN_RESOLVE_HOLD_MS,
  CPU_PLAN_EVENT_STEP_MS,
} from '../constants';
import type {
  Coordinate,
  CpuStatusResponse,
  GameCfg,
  GameEvent,
  GameState,
  TurnCmdType,
  TurnCommand,
  Unit,
} from '../types/api';

type CpuPollResult =
  | { ok: true; status: CpuStatusResponse }
  | { ok: false; reason: 'timeout' | 'idle' };

export interface MatchSceneData {
  roomId: string;
  playerTokens: [string, string];
  isRematch?: boolean;
}

export default class MatchScene extends Phaser.Scene {
  private roomId!: string;
  private playerTokens!: [string, string];
  private gameState!: GameState;
  private gameCfg!: GameCfg;
  private gameCfgPromise!: Promise<GameCfg>;
  private gameStateLoaded = false;
  private gameCfgLoaded = false;
  private terrainObjects: Phaser.GameObjects.GameObject[] = [];
  private occupantObjects: Phaser.GameObjects.GameObject[] = [];
  private unitSpritesById = new Map<number, Phaser.GameObjects.Sprite>();
  private bombGraphicsById = new Map<number, BombGraphics>();
  private softBlockSpritesById = new Map<number, Phaser.GameObjects.Sprite>();
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
  private summaryPanelOpen = false;
  private lifecycleConfirmOpen = false;
  private victoryActionTaken = false;
  // Bumped on 'shutdown'; async callbacks compare against this to detect a torn-down scene.
  private generation = 0;

  constructor() {
    super('MatchScene');
  }

  preload(): void {
    for (const { key, png, json } of SPRITE_MANIFEST) {
      this.load.aseprite(key, `${SPRITE_ASSET_BASE}${png}`, `${SPRITE_ASSET_BASE}${json}`);
    }
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
    this.gameStateLoaded = false;
    this.gameCfgLoaded = false;
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
        establishBoardLayout(state.grid);
        this.renderBoard(state);
        this.tryRenderStageBackground();
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

    this.gameCfgPromise = getMatchConfig();
    this.gameCfgPromise
      .then(cfg => {
        if (gen !== this.generation) {
          return;
        }
        this.gameCfg = cfg;
        this.gameCfgLoaded = true;
        this.tryRenderStageBackground();
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
    if (this.gameStateLoaded && this.gameCfgLoaded) {
      this.turnPanel.update(this.gameState.turn, this.gameCfg.maxTurns, this.gameState.activeTeam);
    }
  }

  private async beginTurn(): Promise<void> {
    const gen = this.generation;
    this.interactionsEnabled = false;
    // Flipped by the detached gameCfgPromise continuation below; read only by the finally.
    let isCpuTurn = false;
    let resolveBannerDone: () => void = () => undefined;
    const bannerDone = new Promise<void>(resolve => {
      resolveBannerDone = resolve;
    });
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

      void this.gameCfgPromise.then(cfg => {
        if (gen !== this.generation || !(cfg.vsCpu && state.activeTeam === 2)) {
          return;
        }
        isCpuTurn = true;
        this.interactionsEnabled = false;
        const pollPromise = this.pollCpuStatus(gen);
        void bannerDone.then(() => {
          if (gen === this.generation) {
            void this.runCpuTurn(pollPromise, gen);
          }
        });
      });

      if (resp.inSuddenDeath) {
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
      resolveBannerDone();
    } catch (err) {
      if (gen !== this.generation) {
        return;
      }
      this.showError(err instanceof Error ? err.message : String(err));
    } finally {
      if (gen === this.generation && !isCpuTurn) {
        this.interactionsEnabled = true;
      }
    }
  }

  // Resolves after `ms`, or immediately on scene shutdown — a bare delayedCall is discarded on
  // shutdown and would otherwise leave its awaiter suspended forever.
  private delayShutdownSafe(ms: number): Promise<void> {
    return new Promise<void>(resolve => {
      this.time.delayedCall(ms, resolve);
      this.events.once('shutdown', () => resolve());
    });
  }

  private async pollCpuStatus(gen: number): Promise<CpuPollResult> {
    const startedAt = Date.now();
    let attempt = 0;
    for (;;) {
      const resp = await consumeCpuStatus();
      if (gen !== this.generation) {
        return { ok: false, reason: 'timeout' };
      }
      if (resp.turnPhase === 'TurnPhaseReady') {
        return { ok: true, status: resp };
      }
      if (resp.turnPhase === 'TurnPhaseIdle') {
        console.error('consumeCpuStatus() returned TurnPhaseIdle during a CPU turn');
        return { ok: false, reason: 'idle' };
      }
      if (Date.now() - startedAt >= CPU_POLL_BUDGET_MS) {
        return { ok: false, reason: 'timeout' };
      }
      const delay = CPU_POLL_BACKOFF_MS[Math.min(attempt, CPU_POLL_BACKOFF_MS.length - 1)]!;
      await this.delayShutdownSafe(delay);
      if (gen !== this.generation) {
        return { ok: false, reason: 'timeout' };
      }
      attempt++;
    }
  }

  private async runCpuTurn(pollPromise: Promise<CpuPollResult>, gen: number): Promise<void> {
    const result = await pollPromise;
    if (gen !== this.generation) {
      return;
    }
    if (!result.ok) {
      await this.recoverFromCpuPollFailure(gen);
      return;
    }

    const { planGameEvents, resolveTurnGameEvents } = result.status;
    const planSnapshot: GameState = {
      ...this.gameState,
      units: [...this.gameState.units],
      bombs: [...this.gameState.bombs],
    };
    for (const [i, event] of planGameEvents.entries()) {
      if (!this.applyGameEvent(event)) {
        return;
      }
      this.applyPlanEventToSnapshot(planSnapshot, event);
      if (i < planGameEvents.length - 1) {
        await this.delayShutdownSafe(CPU_PLAN_EVENT_STEP_MS);
        if (gen !== this.generation) {
          return;
        }
      }
    }
    if (planGameEvents.length > 0) {
      await this.delayShutdownSafe(CPU_PLAN_RESOLVE_HOLD_MS);
      if (gen !== this.generation) {
        return;
      }
    }

    const { ok, done } = playResolveTurnEvents(resolveTurnGameEvents, {
      scene: this,
      gameStateSnapshot: planSnapshot,
      unitSpritesById: this.unitSpritesById,
      bombGraphicsById: this.bombGraphicsById,
      softBlockSpritesById: this.softBlockSpritesById,
      onError: message => this.showError(message),
    });
    if (ok) {
      await done;
      if (gen !== this.generation) {
        return;
      }
    }

    const matchEndedEvent = resolveTurnGameEvents.find(event => event.type === 'matchEnded');
    if (matchEndedEvent) {
      this.handleMatchEnded(matchEndedEvent);
      return;
    }

    await this.resyncFromServer(gen, true, freshState => {
      this.turnPanel.update(freshState.turn, this.gameCfg.maxTurns, freshState.activeTeam);
    });
    if (gen !== this.generation) {
      return;
    }
    await this.beginTurn();
  }

  // Distinguishes a lost animation (turn advanced server-side, re-render silently) from a CPU
  // turn that never completed (surface an error; the match is never force-ended).
  private async recoverFromCpuPollFailure(gen: number): Promise<void> {
    const turnBeforeCpu = this.gameState.turn;
    const freshState = await getMatchState();
    if (gen !== this.generation) {
      return;
    }
    if (freshState.turn !== turnBeforeCpu && freshState.activeTeam === 1) {
      const fadeDone = new Promise<void>(resolve => {
        this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
        this.cameras.main.once('camerafadeoutcomplete', () => resolve());
        this.events.once('shutdown', () => resolve());
      });
      await fadeDone;
      if (gen !== this.generation) {
        return;
      }
      this.renderBoard(freshState);
      this.cameras.main.fadeIn(FADE_MS);
      await this.beginTurn();
      return;
    }
    this.showError('CPU turn did not complete');
  }

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
          bomb.container.setDepth(occupantDepth(position));
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

    const g = this.unitSpritesById.get(unitId);
    if (g) {
      const fromCenter = tileCenter(from);
      const toCenter = tileCenter(to);
      g.setDepth(occupantDepth(to));
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

  private applyPlanEventToSnapshot(snapshot: GameState, event: GameEvent): void {
    switch (event.type) {
      case 'unitMoved': {
        const { unitId, to } = event;
        if (unitId === undefined || !to) {
          return;
        }
        const idx = snapshot.units.findIndex(u => u.id === unitId);
        if (idx !== -1) {
          snapshot.units[idx] = { ...snapshot.units[idx]!, position: to };
        }
        break;
      }
      case 'bombPlaced': {
        const { unitId, bombId, position, countdown, range } = event;
        if (unitId === undefined || bombId === undefined || !position || countdown === undefined) {
          return;
        }
        snapshot.bombs.push({
          id: bombId,
          ownerId: unitId,
          position,
          range: range ?? 0,
          countdown,
        });
        break;
      }
    }
  }

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

  private isBusy(): boolean {
    return this.isSubmitting || !this.interactionsEnabled;
  }

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

  private async handleResetTurn(): Promise<void> {
    if (this.isSubmitting) {
      return;
    }
    const gen = this.generation;
    this.isSubmitting = true;
    this.clearErrors();
    this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
    try {
      await resetTurn();
      if (gen !== this.generation) {
        return;
      }
      const freshState = await getMatchState();
      if (gen !== this.generation) {
        return;
      }
      this.renderBoard(freshState);
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
    let resolveTurnGameEvents: GameEvent[] = [];
    let succeeded = false;
    try {
      try {
        ({ resolveTurnGameEvents } = await resolveTurn());
        if (gen !== this.generation) {
          return;
        }
        const { ok, done } = playResolveTurnEvents(resolveTurnGameEvents, {
          scene: this,
          gameStateSnapshot: this.gameState,
          unitSpritesById: this.unitSpritesById,
          bombGraphicsById: this.bombGraphicsById,
          softBlockSpritesById: this.softBlockSpritesById,
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
      const matchEndedEvent = resolveTurnGameEvents.find(event => event.type === 'matchEnded');
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
    this.interactionsEnabled = false;
    const isVsCpu = this.gameCfgLoaded && this.gameCfg.vsCpu;
    const returnLabel = isVsCpu ? 'Return to Title' : 'Return to Match Settings';
    this.victoryCutscene.play(winnerTeamId, returnLabel, {
      onRematch: () => this.handleRematchClicked(),
      onReturnClicked: () =>
        isVsCpu ? this.handleReturnToTitleClicked() : this.handleReturnToSettingsClicked(),
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

  private async fadeOutAndDeleteMatch(): Promise<void> {
    const fadeDone = new Promise<void>(resolve => {
      this.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
      this.cameras.main.once('camerafadeoutcomplete', () => resolve());
    });
    const deleteDone = deleteMatch().catch(err =>
      console.error('Failed to delete match:', err instanceof Error ? err.message : err)
    );
    await Promise.all([fadeDone, deleteDone]);
  }

  private handleReturnToSettingsClicked(): void {
    if (this.victoryActionTaken) {
      return;
    }
    this.victoryActionTaken = true;
    void this.fadeOutAndDeleteMatch().then(() => {
      // Forward gameCfg so MatchSettingsScene remembers the last match's settings.
      this.scene.start('MatchSettingsScene', {
        gameCfg: this.gameCfg,
      } satisfies MatchSettingsSceneData);
    });
  }

  private handleReturnToTitleClicked(): void {
    if (this.victoryActionTaken) {
      return;
    }
    this.victoryActionTaken = true;
    void this.fadeOutAndDeleteMatch().then(() => {
      this.scene.start('TitleScene');
    });
  }

  private renderBoard(state: GameState): void {
    this.gameState = state;
    this.gameStateLoaded = true;
    renderOccupants(this.boardCtx(), state);
  }

  private tryRenderStageBackground(): void {
    if (!this.gameStateLoaded || !this.gameCfgLoaded) {
      return;
    }
    renderStageBackground(this.boardCtx(), this.gameState.grid, this.gameCfg.stagePreset);
  }

  private boardCtx(): BoardRenderContext {
    return {
      scene: this,
      terrainObjects: this.terrainObjects,
      occupantObjects: this.occupantObjects,
      unitSpritesById: this.unitSpritesById,
      bombGraphicsById: this.bombGraphicsById,
      softBlockSpritesById: this.softBlockSpritesById,
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

  private clearErrors(): void {
    this.errorPanel.clear();
  }
}
