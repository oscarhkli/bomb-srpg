import type Phaser from 'phaser';

// A rectangular region in scene-local coordinates.
export interface PageBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

// Callbacks a Page uses to ask the owning scene to navigate.
export interface SettingsPageNav {
  goNext: () => void;
  goBack: () => void;
  startMatch: () => void;
  exitToTitle: () => void;
}

// A swappable content view; the owning scene keeps the surrounding chrome.
export interface SettingsPage {
  renderHeaderTitle(scene: Phaser.Scene, x: number, y: number): void;
  renderBody(scene: Phaser.Scene, bounds: PageBounds): void;
  renderNav(scene: Phaser.Scene, bounds: PageBounds): void;
  handleBack(): void;
  // Destroys every GameObject this Page created.
  destroy(): void;
}
