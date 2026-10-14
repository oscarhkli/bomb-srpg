import { expect, type Page } from '@playwright/test';
import { activeSceneKey, clickText, waitForInput } from './canvas';

export async function openTitle(page: Page): Promise<void> {
  await page.goto('/');
  await expect.poll(() => activeSceneKey(page)).toBe('TitleScene');
}

export async function startTwoPlayerMatch(page: Page): Promise<void> {
  await openTitle(page);
  await clickText(page, 'Start Game');
  await clickText(page, 'Battle Mode');
  await expect.poll(() => activeSceneKey(page)).toBe('MatchSettingsScene');

  for (let player = 1; player <= 2; player++) {
    await clickText(page, 'Fighter');
    await clickText(page, 'NEXT →');
  }
  await clickText(page, 'Start Match');
  await expect.poll(() => activeSceneKey(page)).toBe('MatchScene');
  await waitForInput(page);
}

export async function startStoryMatch(page: Page): Promise<void> {
  await openTitle(page);
  await clickText(page, 'Start Game');
  await clickText(page, 'Story Mode');
  await expect.poll(() => activeSceneKey(page)).toBe('MatchScene');
  await waitForInput(page);
}
