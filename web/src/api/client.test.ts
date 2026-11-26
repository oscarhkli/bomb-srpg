import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  initRoom,
  initToken,
  ApiError,
  getCatalog,
  createMatchRoom,
  createMatch,
  getMatchState,
  consumeCpuStatus,
  submitTurnCommand,
  resetTurn,
  resolveTurn,
  surrender,
  getMatchConfig,
  getAllowedTiles,
  rematch,
  deleteMatch,
} from './client';
import { makeState, makeCfg } from '../test/fixtures';
import type {
  Archetype,
  GameState,
  GameEvent,
  GameCfg,
  TurnCommand,
  StagePreset,
  Catalog,
  CpuStatusResponse,
  ResolveTurnResponse,
} from './types';

const mockFetch = vi.fn();

function reply(status: number, body?: unknown) {
  if (body === undefined) {
    mockFetch.mockResolvedValueOnce(new Response(null, { status }));
  } else if (typeof body === 'string') {
    mockFetch.mockResolvedValueOnce(
      new Response(body, { status, headers: { 'Content-Type': 'text/plain; charset=utf-8' } })
    );
  } else {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      })
    );
  }
}

async function sent() {
  expect(mockFetch).toHaveBeenCalledOnce();
  const req = (mockFetch.mock.calls[0] as [Request])[0];
  const url = new URL(req.url);
  return {
    method: req.method,
    path: url.pathname,
    query: url.search,
    authorization: req.headers.get('Authorization'),
    contentType: req.headers.get('Content-Type'),
    body: await req.text(),
  };
}

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch);
  mockFetch.mockReset();
  initRoom('test-room-123');
  initToken('test-token-abc');
});

const AUTH = 'Bearer test-token-abc';
const JSON_TYPE = 'application/json';

describe('api.ts', () => {
  describe('getCatalog', () => {
    it('should fetch catalog and return parsed array', async () => {
      const archetypesFixture: Archetype[] = [
        { name: 'Bomber', speed: 3, bombMaxRange: 2, skills: ['Jump'] },
      ];
      const stagePresetsFixture: StagePreset[] = [
        {
          name: 'Plain',
          description: 'A simple open field with no obstacles.',
          width: 9,
          height: 9,
          maxTurns: 60,
        },
      ];
      const fixture: Catalog = { archetypes: archetypesFixture, stagePresets: stagePresetsFixture };
      reply(200, fixture);

      const result = await getCatalog();

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'GET',
        path: '/api/catalog',
        query: '',
        authorization: null,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on non-2xx response', async () => {
      reply(500, 'Internal server error');

      const error = await getCatalog().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(500);
      expect((error as ApiError).message).toBe('Internal server error');
    });
  });

  describe('error and transport behaviour', () => {
    it('should keep a numeric plain-text error body as a string message', async () => {
      reply(500, '123');

      const error = await getCatalog().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(500);
      expect((error as ApiError).message).toBe('123');
    });

    it('should keep the raw text of a numeric error body with a newline', async () => {
      reply(500, '123\n');

      const error = await getCatalog().catch((e: unknown) => e);
      expect((error as ApiError).message).toBe('123\n');
    });

    it('should give an empty message for an empty error body', async () => {
      mockFetch.mockResolvedValueOnce(
        new Response('', { status: 500, headers: { 'Content-Length': '0' } })
      );

      const error = await getCatalog().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(500);
      expect((error as ApiError).message).toBe('');
    });

    it('should reject when the network is down', async () => {
      mockFetch.mockRejectedValueOnce(new TypeError('Failed to fetch'));

      await expect(getCatalog()).rejects.toThrow('Failed to fetch');
    });

    it('should call public routes without a token', async () => {
      vi.resetModules();
      const fresh = await import('./client');
      reply(200, { archetypes: [], stagePresets: [] });
      reply(201, { id: 'room1' });

      await fresh.getCatalog();
      await fresh.createMatchRoom();

      for (const [req] of mockFetch.mock.calls as [Request][]) {
        expect(req.headers.get('Authorization')).toBeNull();
      }
    });
  });

  describe('createMatchRoom', () => {
    it('should POST and return room id', async () => {
      reply(201, { id: 'room1' });

      const result = await createMatchRoom();

      expect(result).toEqual({ id: 'room1' });
      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms',
        query: '',
        authorization: null,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on failure', async () => {
      reply(500, 'Failed to create room');

      const error = await createMatchRoom().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(500);
    });
  });

  describe('createMatch', () => {
    const req = {
      gameCfg: makeCfg({
        stagePreset: 'Plain',
        p1Slots: [{ archetype: 'King', role: 'King' }],
        p2Slots: [{ archetype: 'King', role: 'King' }],
        maxTurns: 10,
      }),
    };

    it('should POST game config and return player tokens', async () => {
      const resp = { success: true, playerTokens: ['token1', 'token2'] };
      reply(201, resp);

      const result = await createMatch(req);

      expect(result).toEqual(resp);
      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms/test-room-123/match',
        query: '',
        authorization: null,
        contentType: JSON_TYPE,
        body: JSON.stringify(req),
      });
    });

    it('should throw ApiError on invalid config', async () => {
      reply(400, 'invalid game config');

      const error = await createMatch(req).catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(400);
      expect((error as ApiError).message).toContain('invalid game config');
    });
  });

  describe('rematch', () => {
    it('should POST game config and return player tokens', async () => {
      const resp = { success: true, playerTokens: ['token1', 'token2'] };
      reply(201, resp);

      const result = await rematch();

      expect(result).toEqual(resp);
      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms/test-room-123/rematch',
        query: '',
        authorization: AUTH,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on invalid config', async () => {
      reply(401, 'invalid token');

      const error = await rematch().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(401);
      expect((error as ApiError).message).toContain('invalid token');
    });
  });

  describe('deleteMatch', () => {
    it('should DELETE with auth and resolve with no content', async () => {
      reply(204, undefined);

      await expect(deleteMatch()).resolves.toBeUndefined();

      expect(await sent()).toEqual({
        method: 'DELETE',
        path: '/api/match-rooms/test-room-123/match',
        query: '',
        authorization: AUTH,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError when match still in progress', async () => {
      reply(409, 'match still in progress');

      const error = await deleteMatch().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(409);
      expect((error as ApiError).message).toBe('match still in progress');
    });
  });

  describe('getMatchState', () => {
    const fixture: GameState = makeState({ activeTeam: 0, grid: [] });

    it('should GET and return game state', async () => {
      reply(200, fixture);

      const result = await getMatchState();

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'GET',
        path: '/api/match-rooms/test-room-123/match/state',
        query: '',
        authorization: null,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError when match not found', async () => {
      reply(404, 'match not found');

      const error = await getMatchState().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(404);
      expect((error as ApiError).message).toBe('match not found');
    });
  });

  describe('consumeCpuStatus', () => {
    const fixture: CpuStatusResponse = {
      turnPhase: 'TurnPhaseReady',
      planGameEvents: [{ type: 'unitMoved', unitId: 1, from: { x: 1, y: 0 }, to: { x: 1, y: 1 } }],
      resolveTurnGameEvents: [
        { type: 'bombExploded', bombId: 1, position: { x: 1, y: 1 }, affectedPositions: [] },
      ],
    };

    it('should POST with auth and return events', async () => {
      reply(200, fixture);

      const result = await consumeCpuStatus();

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms/test-room-123/match/cpu-status/consume',
        query: '',
        authorization: AUTH,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on failure', async () => {
      reply(401, 'invalid player token');

      const error = await consumeCpuStatus().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(401);
    });
  });

  describe('submitTurnCommand', () => {
    const cmdFixture: TurnCommand = { type: 'move', unitId: 1, target: { x: 1, y: 1 } };
    const fixture: GameEvent[] = [
      { type: 'unitMoved', unitId: 1, from: { x: 1, y: 0 }, to: { x: 1, y: 1 } },
    ];

    it('should POST command with auth and return game events', async () => {
      reply(200, fixture);

      const result = await submitTurnCommand(cmdFixture);

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms/test-room-123/match/turn-commands',
        query: '',
        authorization: AUTH,
        contentType: JSON_TYPE,
        body: JSON.stringify(cmdFixture),
      });
    });

    it('should throw ApiError on invalid command', async () => {
      reply(409, 'invalid turn command');

      const error = await submitTurnCommand(cmdFixture).catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(409);
    });
  });

  describe('resetTurn', () => {
    it('should POST with auth and resolve with no content', async () => {
      reply(204, undefined);

      await expect(resetTurn()).resolves.toBeUndefined();

      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms/test-room-123/match/reset',
        query: '',
        authorization: AUTH,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on failure', async () => {
      reply(500, 'internal error');

      await expect(resetTurn()).rejects.toThrow(ApiError);
    });
  });

  describe('resolveTurn', () => {
    const fixture: ResolveTurnResponse = {
      planGameEvents: [{ type: 'unitMoved', unitId: 1, from: { x: 1, y: 0 }, to: { x: 1, y: 1 } }],
      resolveTurnGameEvents: [
        { type: 'bombExploded', bombId: 1, position: { x: 1, y: 1 }, affectedPositions: [] },
      ],
    };

    it('should POST with auth and return events', async () => {
      reply(200, fixture);

      const result = await resolveTurn();

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms/test-room-123/match/resolve',
        query: '',
        authorization: AUTH,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on failure', async () => {
      reply(401, 'invalid player token');

      const error = await resolveTurn().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(401);
    });
  });

  describe('surrender', () => {
    const req = { teamId: 0 };
    const fixture: GameEvent[] = [{ type: 'matchEnded', winnerTeamId: 1 }];

    it('should POST surrender request with auth', async () => {
      reply(200, fixture);

      const result = await surrender(req);

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'POST',
        path: '/api/match-rooms/test-room-123/match/surrender',
        query: '',
        authorization: AUTH,
        contentType: JSON_TYPE,
        body: JSON.stringify(req),
      });
    });

    it('should throw ApiError on invalid team id', async () => {
      reply(400, 'invalid game config');

      await expect(surrender({ teamId: 5 })).rejects.toThrow(ApiError);
    });
  });

  describe('getMatchConfig', () => {
    const fixture: GameCfg = makeCfg({
      vsCpu: false,
      stagePreset: 'Plain',
      p1Slots: [{ archetype: 'King', role: 'King' }],
      p2Slots: [{ archetype: 'King', role: 'King' }],
      maxTurns: 10,
    });

    it('should GET and return game config', async () => {
      reply(200, fixture);

      const result = await getMatchConfig();

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'GET',
        path: '/api/match-rooms/test-room-123/match/config',
        query: '',
        authorization: null,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on failure', async () => {
      reply(404, 'match not found');

      await expect(getMatchConfig()).rejects.toThrow(ApiError);
    });
  });

  describe('getAllowedTiles', () => {
    it('should send unitId and turnCmdType as query params', async () => {
      const fixture = [
        { x: 1, y: 2 },
        { x: 2, y: 3 },
      ];
      reply(200, fixture);

      const result = await getAllowedTiles({ unitId: 1, turnCmdType: 'move' });

      expect(result).toEqual(fixture);
      expect(await sent()).toEqual({
        method: 'GET',
        path: '/api/match-rooms/test-room-123/match/allowed-tiles',
        query: '?unitId=1&turnCmdType=move',
        authorization: null,
        contentType: null,
        body: '',
      });
    });

    it('should throw ApiError on failure', async () => {
      reply(400, 'missing required query parameters');

      const error = await getAllowedTiles({ unitId: 1, turnCmdType: 'move' }).catch(
        (e: unknown) => e
      );
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(400);
    });
  });
});
