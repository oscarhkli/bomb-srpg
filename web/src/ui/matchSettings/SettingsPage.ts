import type Phaser from 'phaser';

// A rectangular region of the scene, in scene-local (not screen) coordinates — used for
// MatchSettingsScene's body/nav regions handed to each Page.
export interface PageBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

// Nav callbacks a Page uses to ask the owning scene to swap to the next/previous Page. The scene
// (not the Page) owns fadeTransition and the pages array/index.
export interface SettingsPageNav {
  goNext: () => void;
  goBack: () => void;
  startMatch: () => void;
  exitToTitle: () => void;
}

// A swappable content view within MatchSettingsScene's body (VISUAL_VOCAB "Page"). The scene's
// chrome (HeaderRegion/NavRegion) stays put while the active Page's render*/destroy methods run.
export interface SettingsPage {
  renderHeaderTitle(scene: Phaser.Scene, x: number, y: number): void;
  renderBody(scene: Phaser.Scene, bounds: PageBounds): void;
  renderNav(scene: Phaser.Scene, bounds: PageBounds): void;
  // What BackButton does while this Page is active (no-op, or navigate to a prior Page).
  handleBack(): void;
  // Destroys every GameObject this Page created (header title + body + nav), so the scene can
  // fadeTransition to a different Page without leaking objects.
  destroy(): void;
}
