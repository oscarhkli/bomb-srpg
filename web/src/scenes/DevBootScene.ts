import Phaser from 'phaser'
import { initRoom, createMatchRoom, createMatch } from '../engine/api'

// Dev-only scene to exercise MatchScene before LoungeScene exists.
// Remove this file and its main.ts registration once LoungeScene is implemented.
export default class DevBootScene extends Phaser.Scene {
  constructor() {
    super('DevBootScene')
  }

  async create(): Promise<void> {
    const { id: roomId } = await createMatchRoom()
    initRoom(roomId)
    const { playerTokens } = await createMatch({
      gameCfg: {
        stagePreset: 'MAP01',
        p1Teams: ['King', 'Fighter'],
        p2Teams: ['King', 'Fighter'],
        maxTurns: 10,
        allowResetTurn: true,
        suddenDeath: false
      }
    })
    this.scene.start('MatchScene', { roomId, playerTokens })
  }
}
