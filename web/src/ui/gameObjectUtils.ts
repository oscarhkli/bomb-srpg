// Destroys every object in the array, then empties it in place — the "clear before redraw"
// idiom repeated across MatchScene/TurnPanel/TurnCommandPanel whenever a panel is rebuilt.
export function destroyAll(objects: { destroy(): void }[]): void {
  objects.forEach(o => o.destroy());
  objects.length = 0;
}

export function colorToCss(color: number): string {
  return `#${color.toString(16).padStart(6, '0')}`;
}
