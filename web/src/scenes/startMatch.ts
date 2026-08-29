import type Phaser from 'phaser';
import { createMatchRoom, initRoom, createMatch } from '../engine/api';
import type ErrorPanel from '../ui/ErrorPanel';
import type { GameCfg } from '../types/api';
import type { MatchSceneData } from './MatchScene';
import { FADE_MS } from '../constants';

export interface StartMatchOptions {
  scene: Phaser.Scene;
  gameCfg: GameCfg;
  isTransitioning: () => boolean;
  setTransitioning: (value: boolean) => void;
  generation: () => number;
  errorPanel: ErrorPanel;
}

export function startMatch(opts: StartMatchOptions): void {
  const { scene, gameCfg, isTransitioning, setTransitioning, generation, errorPanel } = opts;
  if (isTransitioning()) {
    return;
  }
  setTransitioning(true);
  const gen = generation();
  scene.cameras.main.fadeOut(FADE_MS, 0, 0, 0);
  const fadeDone = new Promise<void>(resolve => {
    scene.cameras.main.once('camerafadeoutcomplete', () => resolve());
    // Scene may be torn down before the fade event fires; resolve anyway so this
    // promise can't hang forever (the gen check below still guards the outcome).
    scene.events.once('shutdown', () => resolve());
  });
  const matchResult = createMatchRoom()
    .then(({ id }) => {
      initRoom(id);
      return createMatch({ gameCfg }).then(({ playerTokens }) => ({
        ok: true as const,
        roomId: id,
        playerTokens,
      }));
    })
    .catch((err: unknown) => ({
      ok: false as const,
      message: err instanceof Error ? err.message : String(err),
    }));

  void Promise.all([fadeDone, matchResult]).then(([, result]) => {
    if (gen !== generation()) {
      return;
    }
    if (!result.ok) {
      scene.cameras.main.fadeIn(FADE_MS);
      errorPanel.show(`Failed to create match: ${result.message}`);
      setTransitioning(false);
      return;
    }
    scene.scene.start('MatchScene', {
      roomId: result.roomId,
      playerTokens: result.playerTokens,
    } satisfies MatchSceneData);
  });
}
