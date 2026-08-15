// Shared mockScene accessors/drivers for tests. Centralizes the `mock.results[i].value` casts
// (mockScene.add.graphics()/add.text() return a FRESH instance per call — see setup.ts) and the
// "invoke a scheduled tween/timer callback manually" idiom that recurs across UI/rendering tests.
import { mockScene, createMockText, createMockContainer } from './setup';
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

export function occupantGraphics(index: number): ReturnType<typeof mockScene.add.graphics> {
  return graphicsAt(index);
}

export function allGraphics(): ReturnType<typeof mockScene.add.graphics>[] {
  return mockScene.add.graphics.mock.results.map(
    r => r.value as ReturnType<typeof mockScene.add.graphics>
  );
}

// Last `n` Graphics instances created so far, in creation order — e.g. the 3 buttons a just-opened
// TurnCommandPanel drew (Move/Bomb/Back), or the 1 overlay tile just rendered.
export function lastGraphics(n: number): ReturnType<typeof mockScene.add.graphics>[] {
  return allGraphics().slice(-n);
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

export function textByContent(content: string): ReturnType<typeof mockScene.add.text> {
  const index = textCalls().findIndex(c => c[2] === content);
  if (index === -1) {
    throw new Error(`no text created with content "${content}"`);
  }
  return textAt(index);
}

// Finds and invokes a Text mock's listener for the given pointer event (e.g. 'pointerover').
export function fireTextPointerEvent(
  text: ReturnType<typeof mockScene.add.text>,
  event: string
): void {
  const call = text.on.mock.calls.find(c => c[0] === event);
  if (!call) {
    throw new Error(`no listener registered for "${event}"`);
  }
  (call[1] as () => void)();
}

// Accepts any mock with an `on` spy — Graphics, Sprite, Container, etc. all register listeners
// the same way.
export function pointerDownOf(g: { on: { mock: { calls: unknown[][] } } }): () => void {
  return g.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void;
}

export function clickPointerdown(g: { on: { mock: { calls: unknown[][] } } }): void {
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

// Simulates a fadeOut() finishing by invoking the most recently registered
// 'camerafadeoutcomplete' listener — the mock keeps every registration, unlike a real `.once()`.
export function fireCameraFadeOutComplete(): void {
  const calls = mockScene.cameras.main.once.mock.calls.filter(
    c => c[0] === 'camerafadeoutcomplete'
  );
  const call = calls[calls.length - 1];
  (call?.[1] as (() => void) | undefined)?.();
}

export function makeBombGraphics(): BombGraphics {
  return {
    container: createMockContainer() as never,
    countdownText: createMockText() as never,
  };
}

export function spriteAt(index: number): ReturnType<typeof mockScene.add.sprite> {
  return mockScene.add.sprite.mock.results[index]!.value as ReturnType<typeof mockScene.add.sprite>;
}

export function occupantSprite(index: number): ReturnType<typeof mockScene.add.sprite> {
  return spriteAt(index);
}

export function allSprites(): ReturnType<typeof mockScene.add.sprite>[] {
  return mockScene.add.sprite.mock.results.map(
    r => r.value as ReturnType<typeof mockScene.add.sprite>
  );
}

export function stageBackgroundImage(): ReturnType<typeof mockScene.add.image> {
  return mockScene.add.image.mock.results[0]!.value as ReturnType<typeof mockScene.add.image>;
}
