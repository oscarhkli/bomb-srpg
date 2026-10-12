import { expect, test } from '@playwright/test';
import { activeSceneKey, clickText, waitForText } from './canvas';

test('two-player local match reaches the board', async ({ page }) => {
  await page.goto('/');
  await expect.poll(() => activeSceneKey(page)).toBe('TitleScene');

  await clickText(page, 'Start Game');
  await clickText(page, 'Battle Mode');
  await expect.poll(() => activeSceneKey(page)).toBe('MatchSettingsScene');

  for (let player = 1; player <= 2; player++) {
    await clickText(page, 'Fighter');
    await clickText(page, 'NEXT →');
  }
  await clickText(page, 'Start Match');

  await expect.poll(() => activeSceneKey(page)).toBe('MatchScene');
  await waitForText(page, 'Turn');
});
