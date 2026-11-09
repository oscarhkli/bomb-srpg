import { expect, type APIRequestContext, type Page } from '@playwright/test';
import type { GameState, Unit } from '../src/api/types';
import {
  BUTTON_LABEL_BOMB,
  BUTTON_LABEL_MOVE,
  BUTTON_LABEL_YES,
  DEPTH_ALLOWED_TILE_OVERLAY,
  DEPTH_SUDDEN_DEATH_OVERLAY,
  SPRITE_GROUND_MARGIN,
  TILE_SIZE,
} from '../src/constants';

const DEFAULT_POLL_TIMEOUT_MS = 5_000;

export interface Point {
  x: number;
  y: number;
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

async function waitForTarget(
  page: Page,
  target: Target,
  description: string,
  timeout = DEFAULT_POLL_TIMEOUT_MS
): Promise<Point> {
  let point: Point | undefined;
  await expect
    .poll(async () => (point = await targetCenter(page, target)) !== undefined, {
      message: description,
      timeout,
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

export async function waitForText(
  page: Page,
  label: string,
  timeout = DEFAULT_POLL_TIMEOUT_MS
): Promise<Point> {
  return waitForTarget(
    page,
    { kind: 'text', label },
    `visible text "${label}" in the active scene`,
    timeout
  );
}

export async function clickText(page: Page, label: string): Promise<void> {
  await clickTarget(page, { kind: 'text', label }, `visible text "${label}" in the active scene`);
}

export async function clickUnit(page: Page, unitId: number): Promise<void> {
  await clickTarget(page, { kind: 'unit', id: unitId }, `unit ${unitId} sprite`);
}

export async function clickMoveButton(page: Page): Promise<void> {
  await clickTarget(page, { kind: 'spriteButton', labelKey: BUTTON_LABEL_MOVE }, 'Move button');
}

export async function clickBombButton(page: Page): Promise<void> {
  await clickTarget(page, { kind: 'spriteButton', labelKey: BUTTON_LABEL_BOMB }, 'Bomb button');
}

export async function textVisible(page: Page, label: string): Promise<boolean> {
  return (await targetCenter(page, { kind: 'text', label })) !== undefined;
}

export async function clickConfirmYes(page: Page): Promise<void> {
  await clickTarget(page, { kind: 'spriteButton', labelKey: BUTTON_LABEL_YES }, 'Yes button');
}

export async function unitCenter(page: Page, unitId: number): Promise<Point> {
  return waitForTarget(page, { kind: 'unit', id: unitId }, `unit ${unitId} sprite`);
}

export async function clickAllowedTile(page: Page, index: number): Promise<Point> {
  const tile = await waitForTarget(page, { kind: 'allowedTile', index }, `allowed tile ${index}`);
  await click(page, tile);
  return tile;
}

export async function clickAllowedTileNextTo(page: Page, unitId: number): Promise<Point> {
  const from = await unitCenter(page, unitId);
  const tilePx = (await pixelScale(page)) * TILE_SIZE;
  const tolerance = tilePx / 4;
  await waitForTarget(page, { kind: 'allowedTile', index: 0 }, 'allowed tile 0');
  for (let index = 0; ; index++) {
    const tile = await targetCenter(page, { kind: 'allowedTile', index });
    if (!tile) {
      throw new Error(`no allowed tile one step from unit ${unitId}`);
    }
    const dx = Math.abs(tile.x - from.x);
    const dy = Math.abs(tile.y - from.y);
    const sideways = Math.abs(dx - tilePx) < tolerance && dy < tolerance;
    const vertical = Math.abs(dy - tilePx) < tolerance && dx < tolerance;
    if (sideways || vertical) {
      await click(page, tile);
      return tile;
    }
  }
}

export async function waitForInput(page: Page): Promise<void> {
  await expect
    .poll(
      () =>
        page.evaluate(() => {
          const scene = window.__game?.scene.getScenes(true)[0] as
            { interactionsEnabled?: boolean; isSubmitting?: boolean } | undefined;
          return scene?.interactionsEnabled === true && scene.isSubmitting === false;
        }),
      { message: 'the match to accept input', timeout: 30_000 }
    )
    .toBe(true);
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

export async function matchState(request: APIRequestContext, room: string): Promise<GameState> {
  const res = await request.get(`/api/match-rooms/${room}/match/state`);
  expect(res.ok()).toBe(true);
  return (await res.json()) as GameState;
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

export interface BombSample {
  drawn: number;
  containers: number;
  overlay: boolean;
}

export interface CutsceneSummary {
  before: number;
  atStart: number;
  atEnd: number;
  max: number;
}

export async function watchBombs(page: Page): Promise<void> {
  await page.addInitScript((overlayDepth: number) => {
    interface BombScene {
      scene: { key: string };
      bombGraphicsById: Map<number, unknown>;
      occupantObjects: { type: string }[];
      children: { list: { type: string; depth: number; visible: boolean }[] };
    }
    const w = window as unknown as { __game?: Phaser.Game; __bombWatch: BombSample[] };
    w.__bombWatch = [];
    const tick = (): void => {
      const scene = w.__game?.scene.getScenes(true)[0] as unknown as BombScene | undefined;
      if (scene?.scene.key === 'MatchScene') {
        const sample: BombSample = {
          drawn: scene.bombGraphicsById.size,
          containers: scene.occupantObjects.filter(o => o.type === 'Container').length,
          overlay: scene.children.list.some(
            o => o.type === 'Graphics' && o.depth === overlayDepth && o.visible
          ),
        };
        const last = w.__bombWatch[w.__bombWatch.length - 1];
        if (
          !last ||
          last.drawn !== sample.drawn ||
          last.containers !== sample.containers ||
          last.overlay !== sample.overlay
        ) {
          w.__bombWatch.push(sample);
        }
      }
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  }, DEPTH_SUDDEN_DEATH_OVERLAY);
}

export function readBombWatch(page: Page): Promise<BombSample[]> {
  return page.evaluate(() => (window as unknown as { __bombWatch: BombSample[] }).__bombWatch);
}

export function summariseCutscenes(samples: BombSample[]): CutsceneSummary[] {
  const result: CutsceneSummary[] = [];
  let before = 0;
  let open: CutsceneSummary | undefined;
  for (const sample of samples) {
    if (sample.overlay && !open) {
      open = { before, atStart: sample.drawn, atEnd: sample.drawn, max: sample.drawn };
      result.push(open);
    } else if (sample.overlay && open) {
      open.atEnd = sample.drawn;
      open.max = Math.max(open.max, sample.drawn);
    } else {
      open = undefined;
      before = sample.drawn;
    }
  }
  return result;
}
