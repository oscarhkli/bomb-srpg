import Phaser from 'phaser';
import './style.css';
import MatchScene from './scenes/MatchScene';
import MatchSettingsScene from './scenes/MatchSettingsScene';

new Phaser.Game({
  type: Phaser.AUTO,
  width: 1280,
  height: 720,
  parent: 'app',
  scale: {
    mode: Phaser.Scale.FIT,
    autoCenter: Phaser.Scale.CENTER_BOTH,
  },
  // MatchSettingsScene first = auto-starts on load.
  scene: [MatchSettingsScene, MatchScene],
});
