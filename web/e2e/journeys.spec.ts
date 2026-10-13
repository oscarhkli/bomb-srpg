import { expect, test } from '@playwright/test';
import {
  clickAllowedTile,
  clickConfirmYes,
  clickMoveButton,
  clickSummaryButton,
  clickText,
  clickUnit,
  matchState,
  matchStatus,
  pixelScale,
  roomId,
  unitCenter,
  waitForText,
  waitForTurnBannerToPass,
} from './canvas';
import { startStoryMatch, startTwoPlayerMatch } from './flows';
import { TILE_SIZE } from '../src/constants';

test('a queued move is applied and the turn passes to the other player', async ({
  page,
  request,
}) => {
  await startTwoPlayerMatch(page);
  await waitForTurnBannerToPass(page, 1);
  const room = await roomId(page);
  const unit = (await matchState(request, room)).units.find(u => u.team === 1)!;

  const from = await unitCenter(page, unit.id);
  await clickUnit(page, unit.id);
  await clickMoveButton(page);
  const to = await clickAllowedTile(page, 0);
  await clickConfirmYes(page);

  const tilePx = (await pixelScale(page)) * TILE_SIZE;
  const target = {
    x: unit.position.x + Math.round((to.x - from.x) / tilePx),
    y: unit.position.y + Math.round((to.y - from.y) / tilePx),
  };
  await expect
    .poll(async () => (await matchState(request, room)).units.find(u => u.id === unit.id))
    .toMatchObject({ position: target, hasMoved: true });

  await clickSummaryButton(page);
  await clickText(page, 'End this turn');
  await clickConfirmYes(page);
  await expect
    .poll(async () => matchState(request, room))
    .toMatchObject({ turn: 2, activeTeam: 2 });

  const after = await matchState(request, room);
  expect(after.units.find(u => u.id === unit.id)?.position).toEqual(target);
});

test('surrendering ends the match and returning to settings deletes it', async ({
  page,
  request,
}) => {
  await startTwoPlayerMatch(page);
  await waitForTurnBannerToPass(page, 1);
  const room = await roomId(page);

  await clickSummaryButton(page);
  await clickText(page, 'Surrender');
  await clickConfirmYes(page);
  await waitForText(page, 'Player 2!');

  await clickText(page, 'Return to Match Settings');
  await expect.poll(() => matchStatus(request, room)).toBe(404);
});

test('a VS CPU match plays the CPU turn and returns to the player', async ({ page, request }) => {
  test.setTimeout(90_000);
  await startStoryMatch(page);
  await waitForTurnBannerToPass(page, 1);
  const room = await roomId(page);

  await clickSummaryButton(page);
  await clickText(page, 'End this turn');
  await clickConfirmYes(page);
  await expect
    .poll(async () => matchState(request, room), { timeout: 60_000 })
    .toMatchObject({ turn: 3, activeTeam: 1 });
});
