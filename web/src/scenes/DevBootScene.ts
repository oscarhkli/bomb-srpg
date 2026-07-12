import Phaser from 'phaser';
import { initRoom, createMatchRoom, createMatch } from '../engine/api';

// Dev-only scene to exercise MatchScene before LoungeScene exists.
// Remove this file and its main.ts registration once LoungeScene is implemented.
export default class DevBootScene extends Phaser.Scene {
  constructor() {
    super('DevBootScene');
  }

  async create(): Promise<void> {
    const { id: roomId } = await createMatchRoom();
    initRoom(roomId);
    const { playerTokens } = await createMatch({
      gameCfg: {
        stagePreset: 'MAP03',
        p1Teams: ['King', 'Fighter', 'Witch', 'Bandit', 'Fighter'],
        p2Teams: ['King', 'Fighter', 'Witch', 'Bandit', 'Fighter'],
        maxTurns: 6,
        allowResetTurn: true,
        suddenDeath: true,
      },
    });
    this.scene.start('MatchScene', { roomId, playerTokens });
  }
}
