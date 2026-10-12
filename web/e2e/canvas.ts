import { expect, type Page } from '@playwright/test';

interface Point {
  x: number;
  y: number;
}

export async function activeSceneKey(page: Page): Promise<string | undefined> {
  return page.evaluate(() => window.__game?.scene.getScenes(true)[0]?.scene.key);
}

export async function textCenter(page: Page, label: string): Promise<Point | undefined> {
  return page.evaluate((wanted): Point | undefined => {
    const game = window.__game;
    const scene = game?.scene.getScenes(true)[0];
    const canvas = game?.canvas;
    if (!game || !scene || !canvas || scene.cameras.main.fadeEffect.isRunning) {
      return undefined;
    }
    interface TextLike {
      visible: boolean;
      text: string;
      getBounds(): { centerX: number; centerY: number };
    }
    const match = (scene.children.list as unknown as Partial<TextLike>[]).find(
      (o): o is TextLike => o.visible === true && o.text === wanted
    );
    if (!match) {
      return undefined;
    }
    const rect = canvas.getBoundingClientRect();
    const { centerX, centerY } = match.getBounds();
    return {
      x: rect.left + (centerX * rect.width) / game.scale.width,
      y: rect.top + (centerY * rect.height) / game.scale.height,
    };
  }, label);
}

export async function waitForText(page: Page, label: string): Promise<Point> {
  let point: Point | undefined;
  await expect
    .poll(async () => (point = await textCenter(page, label)) !== undefined, {
      message: `visible text "${label}" in the active scene`,
    })
    .toBe(true);
  if (!point) {
    throw new Error(`text "${label}" disappeared before it could be used`);
  }
  return point;
}

export async function clickText(page: Page, label: string): Promise<void> {
  const { x, y } = await waitForText(page, label);
  await page.mouse.click(x, y);
  // Phaser handles queued input on its next frame, so any fade it starts is not visible yet.
  await page.evaluate(
    () =>
      new Promise<void>(done => requestAnimationFrame(() => requestAnimationFrame(() => done())))
  );
}
