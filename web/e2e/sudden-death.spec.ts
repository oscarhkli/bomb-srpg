import { expect, test } from '@playwright/test';
import {
  clickAllowedTile,
  clickConfirmYes,
  clickMoveButton,
  clickText,
  clickUnit,
  matchState,
  readBombWatch,
  roomId,
  summariseCutscenes,
  teamUnit,
  waitForInput,
  waitForText,
  watchBombs,
} from './canvas';
import { endTurn, startTwoPlayerMatch, startVsCpuRoom } from './flows';
import { MATCH_SUMMARY_BUTTON_LABEL, RESET_BUTTON_LABEL } from '../src/constants';
import type { Bomb, CpuStatusResponse, GameEvent, GameState } from '../src/api/types';

const SYSTEM_OWNER_ID = 0;

test('VS CPU sudden death: hazards are drawn once per turn through the first CPU turn', async ({
  page,
  request,
}) => {
  test.setTimeout(70_000);
  await watchBombs(page);
  const { room, tokens } = await startVsCpuRoom(page, request, 0);

  const first = await matchState(request, room);
  expect(first.inSuddenDeath).toBe(true);
  expect(first.bombs.filter(b => b.ownerId === SYSTEM_OWNER_ID)).toHaveLength(2);

  await endTurn(page);
  await expect
    .poll(async () => matchState(request, room), { timeout: 60_000 })
    .toMatchObject({ turn: 3, activeTeam: 1 });
  await waitForInput(page);

  const last = await matchState(request, room);
  expect(last.bombs.filter(b => b.ownerId === SYSTEM_OWNER_ID)).toHaveLength(6);

  const samples = await readBombWatch(page);
  const cutscenes = summariseCutscenes(samples);
  expect(cutscenes).toHaveLength(3);
  for (const cutscene of cutscenes) {
    expect(cutscene.atEnd - cutscene.atStart).toBe(2);
  }
  const final = samples.at(-1);
  if (!final) {
    throw new Error('the bomb watcher recorded nothing');
  }
  expect(final.drawn).toBe(last.bombs.length);
  expect(final.containers).toBe(final.drawn);

  const consumed = await request.post(`/api/match-rooms/${room}/match/cpu-status/consume`, {
    headers: { Authorization: `Bearer ${tokens[0]}` },
  });
  expect(consumed.ok()).toBe(true);
  expect(((await consumed.json()) as CpuStatusResponse).turnPhase).toBe('TurnPhaseIdle');
});

test('hot-seat sudden death drops the hazards once per turn, after the cutscene starts', async ({
  page,
  request,
}) => {
  test.setTimeout(60_000);
  await watchBombs(page);
  await startTwoPlayerMatch(page, { instantSuddenDeath: true });
  const room = await roomId(page);

  const first = await matchState(request, room);
  expect(first.inSuddenDeath).toBe(true);
  expect(first.bombs).toHaveLength(2);
  expect(first.bombs.every(b => b.ownerId === SYSTEM_OWNER_ID)).toBe(true);
  expect(first.turnStartEvents).toHaveLength(2);
  const bombIds = first.bombs.map(b => b.id);
  for (const evt of first.turnStartEvents) {
    expect(bombIds).toContain(evt.bombId);
  }

  const unit = await teamUnit(request, room, 1);
  await clickUnit(page, unit.id);
  await clickMoveButton(page);
  await clickAllowedTile(page, 0);
  await clickConfirmYes(page);
  await expect
    .poll(async () => (await matchState(request, room)).units.find(u => u.id === unit.id))
    .toMatchObject({ hasMoved: true });
  await clickText(page, MATCH_SUMMARY_BUTTON_LABEL);
  await clickText(page, RESET_BUTTON_LABEL);
  await clickConfirmYes(page);
  await expect
    .poll(async () => (await matchState(request, room)).units.find(u => u.id === unit.id))
    .toMatchObject({ hasMoved: false });
  await waitForInput(page);

  const afterReset = await matchState(request, room);
  expect(afterReset.turnStartEvents).toHaveLength(2);
  expect(afterReset.bombs).toHaveLength(2);
  expect((await readBombWatch(page)).at(-1)).toMatchObject({ drawn: 2, containers: 2 });

  await endTurn(page);
  await expect
    .poll(async () => matchState(request, room))
    .toMatchObject({ turn: 2, activeTeam: 2 });
  await waitForInput(page);

  const second = await matchState(request, room);
  expect(second.bombs).toHaveLength(4);

  const samples = await readBombWatch(page);
  const cutscenes = summariseCutscenes(samples);
  expect(cutscenes).toHaveLength(2);
  expect(cutscenes[0]).toMatchObject({ before: 0, atStart: 0, atEnd: 2 });
  expect(cutscenes[1]).toMatchObject({ before: 2, atStart: 2, atEnd: 4 });
  expect(samples.at(-1)).toMatchObject({ drawn: 4, containers: 4 });
});

test('hot-seat sudden death: a failed resolve and a retry add no extra hazards', async ({
  page,
  request,
}) => {
  test.setTimeout(60_000);
  const failure = 'resolve failed (injected by the test)';
  let failed = false;
  await watchBombs(page);
  await page.route('**/match/resolve', route => {
    if (failed) {
      return route.continue();
    }
    failed = true;
    return route.fulfill({ status: 500, body: failure });
  });
  await startTwoPlayerMatch(page, { instantSuddenDeath: true });
  const room = await roomId(page);

  await endTurn(page);
  await waitForText(page, failure);
  await waitForInput(page);

  const afterFailure = await matchState(request, room);
  expect(afterFailure.turn).toBe(1);
  expect(afterFailure.bombs).toHaveLength(2);
  expect(summariseCutscenes(await readBombWatch(page))).toHaveLength(1);
  expect((await readBombWatch(page)).at(-1)).toMatchObject({ drawn: 2, containers: 2 });

  await endTurn(page);
  await expect
    .poll(async () => matchState(request, room))
    .toMatchObject({ turn: 2, activeTeam: 2 });
  await waitForInput(page);

  expect((await matchState(request, room)).bombs).toHaveLength(4);

  const samples = await readBombWatch(page);
  expect(summariseCutscenes(samples)).toHaveLength(2);
  expect(samples.at(-1)).toMatchObject({ drawn: 4, containers: 4 });
});

test('hot-seat: hazards listed in turnStartEvents are not drawn before their cutscene', async ({
  page,
}) => {
  test.setTimeout(60_000);
  await page.route('**/match/state', async route => {
    const response = await route.fetch();
    const state = (await response.json()) as GameState;
    const free = state.grid.flatMap((row, y) =>
      row.flatMap((tile, x) =>
        tile.type === 'TerrainPlain' && tile.occupantType === 'OccupantNone' ? [{ x, y }] : []
      )
    );
    const bombs: Bomb[] = [9001, 9002].map((id, i) => ({
      id,
      ownerId: SYSTEM_OWNER_ID,
      position: free[i]!,
      range: 2,
      countdown: 5,
    }));
    const turnStartEvents: GameEvent[] = bombs.map(b => ({
      type: 'bombPlaced',
      unitId: SYSTEM_OWNER_ID,
      bombId: b.id,
      position: b.position,
      range: b.range,
      countdown: b.countdown,
    }));
    await route.fulfill({
      response,
      json: { ...state, bombs: [...state.bombs, ...bombs], turnStartEvents },
    });
  });
  await watchBombs(page);
  await startTwoPlayerMatch(page);

  expect((await readBombWatch(page)).at(-1)).toMatchObject({ drawn: 0, containers: 0 });
});
