import Phaser from 'phaser';
import './style.css';
import MatchScene from './scenes/MatchScene';
import MatchSettingsScene from './scenes/MatchSettingsScene';
import TitleScene from './scenes/TitleScene';
import { computeZoom } from './scale';

const game = new Phaser.Game({
  type: Phaser.AUTO,
  width: 640,
  height: 360,
  parent: 'app',
  pixelArt: true,
  scale: {
    mode: Phaser.Scale.NONE,
    zoom: computeZoom(window.innerWidth, window.innerHeight),
    autoCenter: Phaser.Scale.CENTER_BOTH,
  },
  // TitleScene first = auto-starts on load.
  scene: [TitleScene, MatchSettingsScene, MatchScene],
});

function handleResize() {
  const zoom = computeZoom(window.innerWidth, window.innerHeight);
  if (zoom !== game.scale.zoom) {
    game.scale.setZoom(zoom);
  }
}

window.addEventListener('resize', handleResize);
game.events.once(Phaser.Core.Events.DESTROY, () => {
  window.removeEventListener('resize', handleResize);
});
