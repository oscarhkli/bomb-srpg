import Phaser from 'phaser';
import { CANVAS_ZOOM_BREAKPOINT_WIDTH, CANVAS_ZOOM_BREAKPOINT_HEIGHT } from './constants';

export function computeZoom(windowWidth: number, windowHeight: number): number {
  if (windowWidth < CANVAS_ZOOM_BREAKPOINT_WIDTH || windowHeight < CANVAS_ZOOM_BREAKPOINT_HEIGHT) {
    return Phaser.Scale.NO_ZOOM;
  }
  return Phaser.Scale.ZOOM_2X;
}
