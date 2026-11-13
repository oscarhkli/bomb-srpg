import { mockScene, createMockText, createMockContainer } from './setup';
import type { BombGraphics } from '../rendering/resolveTurnPlayer';

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

// Casts add.text's mock.calls, whose inferred call signature is empty.
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

export function pointerDownOf(g: { on: { mock: { calls: unknown[][] } } }): () => void {
  return g.on.mock.calls.find(call => call[0] === 'pointerdown')?.[1] as () => void;
}

export function clickPointerdown(g: { on: { mock: { calls: unknown[][] } } }): void {
  pointerDownOf(g)();
}

export function firePointerEvent(g: { on: { mock: { calls: unknown[][] } } }, event: string): void {
  const call = g.on.mock.calls.find(c => c[0] === event);
  if (!call) {
    throw new Error(`no listener registered for "${event}"`);
  }
  (call[1] as () => void)();
}

export function containerAt(index: number): ReturnType<typeof mockScene.add.container> {
  return mockScene.add.container.mock.results[index]!.value as ReturnType<
    typeof mockScene.add.container
  >;
}

export function allContainers(): ReturnType<typeof mockScene.add.container>[] {
  return mockScene.add.container.mock.results.map(
    r => r.value as ReturnType<typeof mockScene.add.container>
  );
}

export function lastContainers(n: number): ReturnType<typeof mockScene.add.container>[] {
  return allContainers().slice(-n);
}

export function imageAt(index: number): ReturnType<typeof mockScene.add.image> {
  return mockScene.add.image.mock.results[index]!.value as ReturnType<typeof mockScene.add.image>;
}

export function allImages(): ReturnType<typeof mockScene.add.image>[] {
  return mockScene.add.image.mock.results.map(
    r => r.value as ReturnType<typeof mockScene.add.image>
  );
}

export function tweenConfigAt(index: number): unknown {
  return mockScene.tweens.add.mock.calls[index]![0];
}

export function delayedCallAt(delayMs: number): () => void {
  const call = mockScene.time.delayedCall.mock.calls.find(c => c[0] === delayMs);
  return call![1] as () => void;
}

export function fireDelayedCall(delayMs: number): void {
  delayedCallAt(delayMs)();
}

export function drainDelayedCalls(): void {
  let i = 0;
  while (i < mockScene.time.delayedCall.mock.calls.length) {
    const cb = mockScene.time.delayedCall.mock.calls[i]![1] as () => void;
    i++;
    cb();
  }
}

export function fireShutdown(): void {
  const call = mockScene.events.once.mock.calls.find(c => c[0] === 'shutdown');
  (call?.[1] as (() => void) | undefined)?.();
}

export function fireLastShutdownListener(): void {
  const calls = mockScene.events.once.mock.calls.filter(c => c[0] === 'shutdown');
  const call = calls[calls.length - 1];
  (call?.[1] as (() => void) | undefined)?.();
}

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
