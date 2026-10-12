import type Phaser from 'phaser';

declare global {
  interface ImportMetaEnv {
    readonly VITE_E2E?: string;
  }

  interface Window {
    __game?: Phaser.Game;
  }
}
