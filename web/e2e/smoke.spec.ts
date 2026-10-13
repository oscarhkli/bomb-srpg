import { test } from '@playwright/test';
import { waitForText } from './canvas';
import { startTwoPlayerMatch } from './flows';

test('two-player local match reaches the board', async ({ page }) => {
  await startTwoPlayerMatch(page);
  await waitForText(page, 'Turn');
});
