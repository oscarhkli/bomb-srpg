import Phaser from 'phaser';
import './style.css';
import MatchScene from './scenes/MatchScene';
import MatchSettingsScene from './scenes/MatchSettingsScene';
import TitleScene from './scenes/TitleScene';

new Phaser.Game({
  type: Phaser.AUTO,
  width: 1280,
  height: 720,
  parent: 'app',
  scale: {
    mode: Phaser.Scale.FIT,
    autoCenter: Phaser.Scale.CENTER_BOTH,
  },
  // TitleScene first = auto-starts on load.
  scene: [TitleScene, MatchSettingsScene, MatchScene],
});
