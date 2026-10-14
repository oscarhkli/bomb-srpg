import { expect, test } from '@playwright/test';
import {
  clickAllowedTile,
  clickMoveButton,
  clickText,
  clickUnit,
  roomId,
  teamUnit,
  waitForText,
} from './canvas';
import { startTwoPlayerMatch } from './flows';
import { MATCH_SUMMARY_BUTTON_LABEL } from '../src/constants';

const zoomLevels = [
  { name: 'NO_ZOOM', viewport: { width: 1280, height: 720 }, zoom: 1 },
  { name: 'ZOOM_2X', viewport: { width: 1920, height: 1080 }, zoom: 2 },
];

for (const { name, viewport, zoom } of zoomLevels) {
  test(`clicks land on the summary button, units, panel buttons and tiles at ${name}`, async ({
    page,
    request,
  }) => {
    await page.setViewportSize(viewport);
    await startTwoPlayerMatch(page);
    await expect.poll(() => page.evaluate(() => window.__game?.scale.zoom)).toBe(zoom);

    await clickText(page, MATCH_SUMMARY_BUTTON_LABEL);
    await waitForText(page, 'Surrender');
    await clickText(page, 'Back');

    const unit = await teamUnit(request, await roomId(page), 1);
    await clickUnit(page, unit.id);
    await clickMoveButton(page);
    await clickAllowedTile(page, 0);
    await waitForText(page, 'Confirm?');
  });
}
