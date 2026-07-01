import Phaser from 'phaser'
import './style.css'
import MatchScene from './scenes/MatchScene'
import DevBootScene from './scenes/DevBootScene'

new Phaser.Game({
  type: Phaser.AUTO,
  width: 1280,
  height: 720,
  parent: 'app',
  scale: {
    mode: Phaser.Scale.FIT,
    autoCenter: Phaser.Scale.CENTER_BOTH
  },
  // DevBootScene first = auto-starts on load; remove once LoungeScene exists.
  scene: [DevBootScene, MatchScene]
})
