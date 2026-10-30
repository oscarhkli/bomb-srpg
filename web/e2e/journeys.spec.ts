import { expect, test } from '@playwright/test';
import {
  clickAllowedTile,
  clickAllowedTileNextTo,
  clickBombButton,
  clickConfirmYes,
  clickMoveButton,
  clickUnit,
  clickText,
  matchState,
  matchStatus,
  pixelScale,
  roomId,
  teamUnit,
  unitCenter,
  waitForInput,
  waitForText,
} from './canvas';
import { endTurn, startStoryMatch, startTwoPlayerMatch } from './flows';
import { MATCH_SUMMARY_BUTTON_LABEL, TILE_SIZE } from '../src/constants';

test('a queued move is accepted and ending the turn passes play to the other player', async ({
  page,
  request,
}) => {
  await startTwoPlayerMatch(page);
  const room = await roomId(page);
  const unit = await teamUnit(request, room, 1);

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

  await clickText(page, MATCH_SUMMARY_BUTTON_LABEL);
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
  const room = await roomId(page);

  await clickText(page, MATCH_SUMMARY_BUTTON_LABEL);
  await clickText(page, 'Surrender');
  await clickConfirmYes(page);
  await waitForText(page, 'Player 2!');

  await clickText(page, 'Return to Match Settings');
  await expect.poll(() => matchStatus(request, room)).toBe(404);
});

test('a VS CPU match plays the CPU turn and hands control back to the player', async ({
  page,
  request,
}) => {
  test.setTimeout(90_000);
  await startStoryMatch(page);
  const room = await roomId(page);

  await clickText(page, MATCH_SUMMARY_BUTTON_LABEL);
  await clickText(page, 'End this turn');
  await clickConfirmYes(page);
  await expect
    .poll(async () => matchState(request, room), { timeout: 60_000 })
    .toMatchObject({ turn: 3, activeTeam: 1 });

  await waitForInput(page);
});

test("a hot-seat match ends without surrender when a King's bomb goes off", async ({
  page,
  request,
}) => {
  test.setTimeout(80_000);
  await startTwoPlayerMatch(page);
  const room = await roomId(page);
  const state = await matchState(request, room);
  expect(state.inSuddenDeath).toBe(false);
  const king = state.units.find(u => u.team === 1 && u.type === 'King');
  if (!king) {
    throw new Error('team 1 has no King');
  }

  await clickUnit(page, king.id);
  await clickBombButton(page);
  await clickAllowedTileNextTo(page, king.id);
  await clickConfirmYes(page);
  await expect.poll(async () => (await matchState(request, room)).bombs).toHaveLength(1);
  const [placed] = (await matchState(request, room)).bombs;
  if (!placed) {
    throw new Error('the King bomb was not placed');
  }

  for (let turn = 1; turn < placed.countdown; turn++) {
    await endTurn(page);
    await expect
      .poll(async () => matchState(request, room))
      .toMatchObject({ turn: turn + 1, activeTeam: turn % 2 === 0 ? 1 : 2 });
    await waitForInput(page);
  }

  const beforeBlast = await matchState(request, room);
  expect(beforeBlast.inSuddenDeath).toBe(false);
  expect(beforeBlast.bombs).toMatchObject([{ countdown: 1 }]);
  expect(beforeBlast.units.find(u => u.id === king.id)?.hp).toBeGreaterThan(0);

  await endTurn(page);
  await waitForText(page, 'Player 2!', 30_000);

  const after = await matchState(request, room);
  expect(after.turn).toBe(5);
  expect(after.units.find(u => u.id === king.id)?.hp).toBe(0);
  expect(after.units.find(u => u.team === 2 && u.type === 'King')?.hp).toBeGreaterThan(0);
});
