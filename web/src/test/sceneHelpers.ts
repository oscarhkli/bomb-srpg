// Shared mockScene accessors/drivers for tests. Centralizes the `mock.results[i].value` casts
// (mockScene.add.graphics()/add.text() return a FRESH instance per call — see setup.ts) and the
// "invoke a scheduled tween/timer callback manually" idiom that recurs across UI/rendering tests.
import { mockScene, createMockGraphics, createMockText, createMockContainer } from './setup';
import type { BombGraphics } from '../rendering/resolveTurnPlayer';

// Drains the microtask queue `times` times — enough for a chain of already-resolved mock
// promises (`.then`/`await` hops) to settle. Cheap since nothing here is a real timer.
export async function flush(times = 15): Promise<void> {
  for (let i = 0; i < times; i++) {
    await Promise.resolve();
  }
}

export function graphicsAt(index: number): ReturnType<typeof mockScene.add.graphics> {
  return mockScene.add.graphics.mock.results[index]!.value as ReturnType<
    typeof mockScene.add.graphics
  >;
}

export function firstGraphics(): ReturnType<typeof mockScene.add.graphics> {
  return graphicsAt(0);
}

// The grid is always the first Graphics created; occupants (units/softBlocks/bombs) follow in
// array order, so occupant index 0 is overall results[1].
export function occupantGraphics(index: number): ReturnType<typeof mockScene.add.graphics> {
  return graphicsAt(index + 1);
}

export function allGraphics(): ReturnType<typeof mockScene.add.graphics>[] {
  return mockScene.add.graphics.mock.results.map(
    r => r.value as ReturnType<typeof mockScene.add.graphics>
  );
}

export function textAt(index: number): ReturnType<typeof mockScene.add.text> {
  return mockScene.add.text.mock.results[index]!.value as ReturnType<typeof mockScene.add.text>;
}

export function firstText(): ReturnType<typeof mockScene.add.text> {
  return textAt(0);
}

export function allTexts(): ReturnType<typeof mockScene.add.text>[] {
  return mockScene.add.text.mock.results.map(r => r.value as ReturnType<typeof mockScene.add.text>);
}

// mockScene.add.text's mock.calls infer a `[]` call-signature (from its no-arg factory
// implementation), so raw index access needs a cast — this helper centralizes it.
export function textCalls(): [number, number, string, ...unknown[]][] {
  return mockScene.add.text.mock.calls as unknown as [number, number, string, ...unknown[]][];
}

export function errorTextByMessage(message: string): ReturnType<typeof mockScene.add.text> {
  const index = textCalls().findIndex(c => c[2] === message);
  return textAt(index);
}

export function pointerDownOf(g: ReturnType<typeof mockScene.add.graphics>): () => void {
  return g.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void;
}

export function clickPointerdown(g: ReturnType<typeof mockScene.add.graphics>): void {
  pointerDownOf(g)();
}

export function tweenConfigAt(index: number): unknown {
  return mockScene.tweens.add.mock.calls[index]![0];
}

// Returns the callback scheduled at delayMs (does not invoke it).
export function delayedCallAt(delayMs: number): () => void {
  const call = mockScene.time.delayedCall.mock.calls.find(c => c[0] === delayMs);
  return call![1] as () => void;
}

// Finds and invokes the callback scheduled at delayMs.
export function fireDelayedCall(delayMs: number): void {
  delayedCallAt(delayMs)();
}

// Finds and invokes the scene's 'shutdown' listener registered via this.events.once(...),
// simulating the scene being torn down mid-flight.
export function fireShutdown(): void {
  const call = mockScene.events.once.mock.calls.find(c => c[0] === 'shutdown');
  (call?.[1] as (() => void) | undefined)?.();
}

// Finds and invokes the camera's 'camerafadeoutcomplete' listener registered via
// this.cameras.main.once(...), simulating a fadeOut() finishing.
export function fireCameraFadeOutComplete(): void {
  const call = mockScene.cameras.main.once.mock.calls.find(c => c[0] === 'camerafadeoutcomplete');
  (call?.[1] as (() => void) | undefined)?.();
}

export function makeBombGraphics(): BombGraphics {
  return {
    container: createMockContainer() as never,
    circle: createMockGraphics() as never,
    countdownText: createMockText() as never,
  };
}
