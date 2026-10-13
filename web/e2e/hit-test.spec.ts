import { expect, test } from '@playwright/test';
import {
  clickAllowedTile,
  clickMoveButton,
  openSummaryPanel,
  openUnitPanel,
  roomId,
  teamUnit,
  waitForText,
} from './canvas';
import { startTwoPlayerMatch } from './flows';

const zoomLevels = [
  { name: 'NO_ZOOM', viewport: { width: 1280, height: 720 }, zoom: 1 },
  { name: 'ZOOM_2X', viewport: { width: 1920, height: 1080 }, zoom: 2 },
];

for (const { name, viewport, zoom } of zoomLevels) {
  test.describe(`at ${name}`, () => {
    test.use({ viewport });

    test.beforeEach(async ({ page }) => {
      await startTwoPlayerMatch(page);
      await expect.poll(() => page.evaluate(() => window.__game?.scale.zoom)).toBe(zoom);
    });

    test('clicks land on board units, allowed tiles and panel buttons', async ({
      page,
      request,
    }) => {
      const unit = await teamUnit(request, await roomId(page), 1);
      await openUnitPanel(page, unit.id);
      await clickMoveButton(page);
      await clickAllowedTile(page, 0);
      await waitForText(page, 'Confirm?');
    });

    test('clicks land on the match summary button', async ({ page }) => {
      await openSummaryPanel(page);
      await waitForText(page, 'Surrender');
    });
  });
}
