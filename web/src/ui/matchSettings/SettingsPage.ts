import type Phaser from 'phaser';

export interface PageBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface SettingsPageNav {
  goNext: () => void;
  goBack: () => void;
  startMatch: () => void;
  exitToTitle: () => void;
}

export interface SettingsPage {
  renderHeaderTitle(scene: Phaser.Scene, x: number, y: number): void;
  renderBody(scene: Phaser.Scene, bounds: PageBounds): void;
  renderNav(scene: Phaser.Scene, bounds: PageBounds): void;
  handleBack(): void;
  destroy(): void;
}
