import Phaser from 'phaser';
import './style.css';
import MatchScene from './scenes/MatchScene';
import MatchSettingsScene from './scenes/MatchSettingsScene';
import TitleScene from './scenes/TitleScene';

new Phaser.Game({
  type: Phaser.AUTO,
  width: 640,
  height: 360,
  parent: 'app',
  pixelArt: true,
  scale: {
    mode: Phaser.Scale.NONE,
    zoom: Phaser.Scale.ZOOM_2X,
    autoCenter: Phaser.Scale.CENTER_BOTH,
  },
  // TitleScene first = auto-starts on load.
  scene: [TitleScene, MatchSettingsScene, MatchScene],
});
