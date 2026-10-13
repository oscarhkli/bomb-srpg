import { expect, test } from '@playwright/test';
import {
  clickAllowedTile,
  clickMoveButton,
  clickSummaryButton,
  clickUnit,
  matchState,
  roomId,
  waitForText,
  waitForTurnBannerToPass,
} from './canvas';
import { startTwoPlayerMatch } from './flows';

const zoomLevels = [
  { name: 'NO_ZOOM', viewport: { width: 1280, height: 720 }, zoom: 1 },
  { name: 'ZOOM_2X', viewport: { width: 1920, height: 1080 }, zoom: 2 },
];

for (const { name, viewport, zoom } of zoomLevels) {
  test.describe(`at ${name}`, () => {
    test.use({ viewport });

    test('clicks land on board units, allowed tiles and panel buttons', async ({
      page,
      request,
    }) => {
      await startTwoPlayerMatch(page);
      await waitForTurnBannerToPass(page, 1);

      const unit = (await matchState(request, await roomId(page))).units.find(u => u.team === 1)!;
      await clickUnit(page, unit.id);
      await clickMoveButton(page);
      await clickAllowedTile(page, 0);
      await waitForText(page, 'Confirm?');
      await expect.poll(() => page.evaluate(() => window.__game?.scale.zoom)).toBe(zoom);
    });

    test('clicks land on the match summary button', async ({ page }) => {
      await startTwoPlayerMatch(page);
      await waitForTurnBannerToPass(page, 1);
      await clickSummaryButton(page);
      await waitForText(page, 'Surrender');
    });
  });
}
