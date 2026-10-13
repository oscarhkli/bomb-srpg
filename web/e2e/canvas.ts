import { expect, type APIRequestContext, type Page } from '@playwright/test';
import {
  BUTTON_LABEL_MOVE,
  BUTTON_LABEL_YES,
  DEPTH_ALLOWED_TILE_OVERLAY,
  MATCH_SUMMARY_BUTTON_LABEL,
  SPRITE_GROUND_MARGIN,
  TILE_SIZE,
} from '../src/constants';

export interface Point {
  x: number;
  y: number;
}

export interface Unit {
  id: number;
  team: number;
  position: Point;
  hasMoved: boolean;
}

export interface MatchState {
  turn: number;
  activeTeam: number;
  units: Unit[];
}

type Target =
  | { kind: 'text'; label: string }
  | { kind: 'spriteButton'; labelKey: string }
  | { kind: 'unit'; id: number }
  | { kind: 'allowedTile'; index: number };

interface TargetConstants {
  overlayDepth: number;
  tileSize: number;
  groundMargin: number;
}

const TARGET_CONSTANTS: TargetConstants = {
  overlayDepth: DEPTH_ALLOWED_TILE_OVERLAY,
  tileSize: TILE_SIZE,
  groundMargin: SPRITE_GROUND_MARGIN,
};

export async function activeSceneKey(page: Page): Promise<string | undefined> {
  return page.evaluate(() => window.__game?.scene.getScenes(true)[0]?.scene.key);
}

async function targetCenter(page: Page, target: Target): Promise<Point | undefined> {
  return page.evaluate(
    ([wanted, k]): Point | undefined => {
      const game = window.__game;
      const scene = game?.scene.getScenes(true)[0];
      const canvas = game?.canvas;
      if (!game || !scene || !canvas || scene.cameras.main.fadeEffect.isRunning) {
        return undefined;
      }
      interface Shown {
        visible: boolean;
      }
      interface TextLike extends Shown {
        text: string;
        getBounds(): { centerX: number; centerY: number };
      }
      interface ButtonLike extends Shown {
        list: { texture?: { key: string } }[];
        input: { enabled: boolean } | null;
        getBounds(): { centerX: number; centerY: number };
      }
      interface OverlayLike extends Shown {
        type: string;
        depth: number;
        x: number;
        y: number;
        input: { enabled: boolean; hitArea: { centerX: number; centerY: number } } | null;
      }
      interface UnitSprite {
        x: number;
        y: number;
      }
      interface MatchSceneInternals {
        unitSpritesById?: Map<number, UnitSprite>;
      }

      let center: Point | undefined;
      if (wanted.kind === 'text') {
        const match = (scene.children.list as unknown as Partial<TextLike>[]).find(
          (o): o is TextLike => o.visible === true && o.text === wanted.label
        );
        if (match) {
          const { centerX, centerY } = match.getBounds();
          center = { x: centerX, y: centerY };
        }
      } else if (wanted.kind === 'spriteButton') {
        const match = (scene.children.list as unknown as Partial<ButtonLike>[]).find(
          (o): o is ButtonLike =>
            o.visible === true &&
            o.input?.enabled === true &&
            Array.isArray(o.list) &&
            o.list.some(child => child.texture?.key === wanted.labelKey)
        );
        if (match) {
          const { centerX, centerY } = match.getBounds();
          center = { x: centerX, y: centerY };
        }
      } else if (wanted.kind === 'unit') {
        const sprite = (scene as unknown as MatchSceneInternals).unitSpritesById?.get(wanted.id);
        if (sprite) {
          center = { x: sprite.x, y: sprite.y - k.groundMargin - k.tileSize / 2 };
        }
      } else {
        const tile = (scene.children.list as unknown as Partial<OverlayLike>[]).filter(
          (o): o is OverlayLike =>
            o.visible === true &&
            o.type === 'Graphics' &&
            o.depth === k.overlayDepth &&
            o.input?.enabled === true
        )[wanted.index];
        if (tile) {
          center = {
            x: tile.x + (tile.input?.hitArea.centerX ?? 0),
            y: tile.y + (tile.input?.hitArea.centerY ?? 0),
          };
        }
      }
      if (!center) {
        return undefined;
      }
      const rect = canvas.getBoundingClientRect();
      return {
        x: rect.left + (center.x * rect.width) / game.scale.width,
        y: rect.top + (center.y * rect.height) / game.scale.height,
      };
    },
    [target, TARGET_CONSTANTS] as const
  );
}

async function waitForTarget(page: Page, target: Target, description: string): Promise<Point> {
  let point: Point | undefined;
  await expect
    .poll(async () => (point = await targetCenter(page, target)) !== undefined, {
      message: description,
    })
    .toBe(true);
  if (!point) {
    throw new Error(`${description} disappeared before it could be used`);
  }
  return point;
}

function twoFrames(page: Page): Promise<void> {
  return page.evaluate(
    () =>
      new Promise<void>(done => requestAnimationFrame(() => requestAnimationFrame(() => done())))
  );
}

async function click(page: Page, point: Point): Promise<void> {
  await page.mouse.move(point.x, point.y);
  await twoFrames(page);
  await page.mouse.click(point.x, point.y);
  await twoFrames(page);
}

async function clickTarget(page: Page, target: Target, description: string): Promise<void> {
  await click(page, await waitForTarget(page, target, description));
}

export async function waitForText(page: Page, label: string): Promise<Point> {
  return waitForTarget(
    page,
    { kind: 'text', label },
    `visible text "${label}" in the active scene`
  );
}

export async function clickText(page: Page, label: string): Promise<void> {
  await clickTarget(page, { kind: 'text', label }, `visible text "${label}" in the active scene`);
}

export async function clickUnit(page: Page, unitId: number): Promise<void> {
  await clickTarget(page, { kind: 'unit', id: unitId }, `unit ${unitId} sprite`);
}

async function clickUntilVisible(
  page: Page,
  click: () => Promise<void>,
  next: Target
): Promise<void> {
  // The game ignores input while it animates, so retry until the next control appears.
  await expect(async () => {
    await click();
    expect(await targetCenter(page, next)).toBeDefined();
  }).toPass({ timeout: 15_000 });
}

export async function clickMoveButton(page: Page): Promise<void> {
  await clickUntilVisible(
    page,
    () => clickTarget(page, { kind: 'spriteButton', labelKey: BUTTON_LABEL_MOVE }, 'Move button'),
    { kind: 'allowedTile', index: 0 }
  );
}

export async function clickLifecycleOption(page: Page, label: string): Promise<void> {
  await clickUntilVisible(page, () => clickText(page, label), {
    kind: 'spriteButton',
    labelKey: BUTTON_LABEL_YES,
  });
}

export async function clickConfirmYes(page: Page): Promise<void> {
  await clickTarget(page, { kind: 'spriteButton', labelKey: BUTTON_LABEL_YES }, 'Yes button');
}

export async function openSummaryPanel(page: Page): Promise<void> {
  await clickUntilVisible(page, () => clickText(page, MATCH_SUMMARY_BUTTON_LABEL), {
    kind: 'text',
    label: 'Surrender',
  });
}

export async function openUnitPanel(page: Page, unitId: number): Promise<void> {
  await clickUntilVisible(page, () => clickUnit(page, unitId), {
    kind: 'spriteButton',
    labelKey: BUTTON_LABEL_MOVE,
  });
}

export async function unitCenter(page: Page, unitId: number): Promise<Point> {
  return waitForTarget(page, { kind: 'unit', id: unitId }, `unit ${unitId} sprite`);
}

export async function clickAllowedTile(page: Page, index: number): Promise<Point> {
  let tile: Point | undefined;
  await clickUntilVisible(
    page,
    async () => {
      tile = await waitForTarget(page, { kind: 'allowedTile', index }, `allowed tile ${index}`);
      await click(page, tile);
    },
    { kind: 'spriteButton', labelKey: BUTTON_LABEL_YES }
  );
  if (!tile) {
    throw new Error(`allowed tile ${index} was never clicked`);
  }
  return tile;
}

export function pixelScale(page: Page): Promise<number> {
  return page.evaluate(() => {
    const game = window.__game;
    const canvas = game?.canvas;
    if (!game || !canvas) {
      throw new Error('game is not running');
    }
    return canvas.getBoundingClientRect().width / game.scale.width;
  });
}

export async function roomId(page: Page): Promise<string> {
  let id: string | undefined;
  await expect
    .poll(
      async () =>
        (id = await page.evaluate(() => {
          for (const entry of performance.getEntriesByType('resource')) {
            const found = /\/api\/match-rooms\/([^/]+)\/match/.exec(entry.name);
            if (found) {
              return found[1];
            }
          }
          return undefined;
        })) !== undefined,
      { message: 'a match-room request from the page' }
    )
    .toBe(true);
  if (!id) {
    throw new Error('no match-room request seen by the page');
  }
  return id;
}

export async function matchState(request: APIRequestContext, room: string): Promise<MatchState> {
  const res = await request.get(`/api/match-rooms/${room}/match/state`);
  expect(res.ok()).toBe(true);
  return (await res.json()) as MatchState;
}

export async function matchStatus(request: APIRequestContext, room: string): Promise<number> {
  return (await request.get(`/api/match-rooms/${room}/match/state`)).status();
}

export async function teamUnit(
  request: APIRequestContext,
  room: string,
  team: number
): Promise<Unit> {
  const unit = (await matchState(request, room)).units.find(u => u.team === team);
  if (!unit) {
    throw new Error(`no unit for team ${team} in room ${room}`);
  }
  return unit;
}
