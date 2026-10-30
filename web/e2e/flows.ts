import { expect, type APIRequestContext, type Page } from '@playwright/test';
import { activeSceneKey, clickConfirmYes, clickText, textVisible, waitForInput } from './canvas';
import { MATCH_SUMMARY_BUTTON_LABEL, MAX_TURNS_ARROW_RIGHT_LABEL } from '../src/constants';

export async function openTitle(page: Page): Promise<void> {
  await page.goto('/');
  await expect.poll(() => activeSceneKey(page)).toBe('TitleScene');
}

const MAX_TURNS_ARROW_CLICKS = 8;
const SUDDEN_DEATH_LABEL = '💣';

export async function selectInstantSuddenDeath(page: Page): Promise<void> {
  for (let clicks = 0; !(await textVisible(page, SUDDEN_DEATH_LABEL)); clicks++) {
    if (clicks >= MAX_TURNS_ARROW_CLICKS) {
      throw new Error('the max-turns control never reached the bomb emoji');
    }
    await clickText(page, MAX_TURNS_ARROW_RIGHT_LABEL);
  }
}

export async function startTwoPlayerMatch(
  page: Page,
  options: { instantSuddenDeath?: boolean } = {}
): Promise<void> {
  await openTitle(page);
  await clickText(page, 'Start Game');
  await clickText(page, 'Battle Mode');
  await expect.poll(() => activeSceneKey(page)).toBe('MatchSettingsScene');

  for (let player = 1; player <= 2; player++) {
    await clickText(page, 'Fighter');
    await clickText(page, 'NEXT →');
  }
  if (options.instantSuddenDeath) {
    await selectInstantSuddenDeath(page);
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

export async function endTurn(page: Page): Promise<void> {
  await clickText(page, MATCH_SUMMARY_BUTTON_LABEL);
  await clickText(page, 'End this turn');
  await clickConfirmYes(page);
}

export async function startVsCpuRoom(
  page: Page,
  request: APIRequestContext,
  maxTurns: number
): Promise<{ room: string; tokens: string[] }> {
  await openTitle(page);
  const created = await request.post('/api/match-rooms');
  expect(created.status()).toBe(201);
  const { id: room } = (await created.json()) as { id: string };

  const slots = [
    { archetype: 'King', role: 'King' },
    { archetype: 'Fighter', role: 'Normal' },
  ];
  const started = await request.post(`/api/match-rooms/${room}/match`, {
    data: {
      gameCfg: {
        vsCpu: true,
        stagePreset: 'Plain',
        p1Slots: slots,
        p2Slots: slots,
        maxTurns,
        allowResetTurn: true,
      },
    },
  });
  expect(started.status()).toBe(201);
  const { playerTokens: tokens } = (await started.json()) as { playerTokens: string[] };

  await page.evaluate(
    ([roomId, playerTokens]) => {
      const game = window.__game;
      if (!game) {
        throw new Error('game is not running');
      }
      game.scene.stop('TitleScene');
      game.scene.start('MatchScene', { roomId, playerTokens });
    },
    [room, tokens] as const
  );
  await expect.poll(() => activeSceneKey(page)).toBe('MatchScene');
  await waitForInput(page);
  return { room, tokens };
}
