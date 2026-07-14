import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  initRoom,
  initToken,
  ApiError,
  getArchetypes,
  createMatchRoom,
  createMatch,
  getMatchState,
  startTurn,
  submitTurnCommand,
  resetTurn,
  resolveTurn,
  surrender,
  getMatchConfig,
  getAllowedTiles,
  rematch,
} from './api';
import { makeState, makeCfg, makeBombPlacedEvent } from '../test/fixtures';
import type {
  Archetype,
  GameState,
  GameEvent,
  GameCfg,
  TurnCommand,
  StartTurnResponse,
} from '../types/api';

const mockFetch = vi.fn();

function mockOk<T>(status: number, body: T) {
  mockFetch.mockResolvedValueOnce({
    ok: true,
    status,
    json: () => Promise.resolve(body),
  });
}

function mockErr(status: number, message: string) {
  mockFetch.mockResolvedValueOnce({
    ok: false,
    status,
    text: () => Promise.resolve(message),
  });
}

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch);
  mockFetch.mockClear();
  initRoom('test-room-123');
  initToken('test-token-abc');
});

describe('api.ts', () => {
  describe('getArchetypes', () => {
    it('should fetch archetypes and return parsed array', async () => {
      const fixture: Archetype[] = [
        { name: 'Bomber', speed: 3, bombMaxRange: 2, skills: ['Jump'] },
      ];
      mockOk(200, fixture);

      const result = await getArchetypes();

      expect(result).toEqual(fixture);
      expect(mockFetch).toHaveBeenCalledOnce();
      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/archetypes'));
    });

    it('should throw ApiError on non-2xx response', async () => {
      mockErr(500, 'Internal server error');

      const error = await getArchetypes().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(500);
      expect((error as ApiError).message).toBe('Internal server error');
    });
  });

  describe('createMatchRoom', () => {
    it('should POST and return room id', async () => {
      mockOk(201, { id: 'room1' });

      const result = await createMatchRoom();

      expect(result).toEqual({ id: 'room1' });
      expect(mockFetch).toHaveBeenCalledOnce();
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringMatching(/\/api\/match-rooms$/),
        expect.objectContaining({ method: 'POST' })
      );
    });

    it('should throw ApiError on failure', async () => {
      mockErr(500, 'Failed to create room');

      const error = await createMatchRoom().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(500);
    });
  });

  describe('createMatch', () => {
    const req = {
      gameCfg: makeCfg({
        stagePreset: 'MAP01',
        p1Teams: ['King'],
        p2Teams: ['King'],
        maxTurns: 10,
      }),
    };

    it('should POST game config and return player tokens', async () => {
      const resp = { success: true, playerTokens: ['token1', 'token2'] };
      mockOk(201, resp);

      const result = await createMatch(req);

      expect(result).toEqual(resp);
      const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/api/match-rooms/test-room-123/match');
      expect(options.method).toBe('POST');
      expect(options.body).toBe(JSON.stringify(req));
    });

    it('should throw ApiError on invalid config', async () => {
      mockErr(400, 'invalid game config');

      const error = await createMatch(req).catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(400);
      expect((error as ApiError).message).toContain('invalid game config');
    });
  });

  describe('rematch', () => {
    it('should POST game config and return player tokens', async () => {
      const resp = { success: true, playerTokens: ['token1', 'token2'] };
      mockOk(201, resp);

      const result = await rematch();

      expect(result).toEqual(resp);
      const [url, options] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/api/match-rooms/test-room-123/rematch');
      expect(options.method).toBe('POST');
    });

    it('should throw ApiError on invalid config', async () => {
      mockErr(401, 'invalid token');

      const error = await rematch().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(401);
      expect((error as ApiError).message).toContain('invalid token');
    });
  });

  describe('getMatchState', () => {
    const fixture: GameState = makeState({ activeTeam: 0, grid: [] });

    it('should GET and return game state', async () => {
      mockOk(200, fixture);

      const result = await getMatchState();

      expect(result).toEqual(fixture);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/match-rooms/test-room-123/match/state')
      );
    });

    it('should throw ApiError when match not found', async () => {
      mockErr(404, 'match not found');

      const error = await getMatchState().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(404);
      expect((error as ApiError).message).toBe('match not found');
    });
  });

  describe('startTurn', () => {
    const gameEventFixture: GameEvent[] = [
      makeBombPlacedEvent({ position: { x: 1, y: 0 }, range: 1, countdown: 5 }),
    ];
    const fixture: StartTurnResponse = {
      inSuddenDeath: false,
      gameEvents: gameEventFixture,
    };

    it('should POST with Authorization header and return startTurnResponse', async () => {
      mockOk(200, fixture);

      const result = await startTurn();

      expect(result).toEqual(fixture);
      const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(options.headers).toHaveProperty('Authorization', 'Bearer test-token-abc');
    });

    it('should throw ApiError on invalid token', async () => {
      mockErr(401, 'invalid player token');

      const error = await startTurn().catch((e: unknown) => e);
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
      mockOk(200, fixture);

      const result = await submitTurnCommand(cmdFixture);

      expect(result).toEqual(fixture);
      const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(options.method).toBe('POST');
      expect(options.headers).toHaveProperty('Authorization', 'Bearer test-token-abc');
      expect(options.body).toBe(JSON.stringify(cmdFixture));
    });

    it('should throw ApiError on invalid command', async () => {
      mockErr(409, 'invalid turn command');

      const error = await submitTurnCommand(cmdFixture).catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(409);
    });
  });

  describe('resetTurn', () => {
    const fixture: GameState = makeState({ activeTeam: 0, grid: [] });

    it('should POST with auth and return game state', async () => {
      mockOk(200, fixture);

      const result = await resetTurn();

      expect(result).toEqual(fixture);
      const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(options.headers).toHaveProperty('Authorization');
    });

    it('should throw ApiError on failure', async () => {
      mockErr(500, 'internal error');

      await expect(resetTurn()).rejects.toThrow(ApiError);
    });
  });

  describe('resolveTurn', () => {
    const fixture: GameEvent[] = [{ type: 'bombExploded', bombId: 1 }];

    it('should POST with auth and return events', async () => {
      mockOk(200, fixture);

      const result = await resolveTurn();

      expect(result).toEqual(fixture);
      const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(options.headers).toHaveProperty('Authorization');
    });

    it('should throw ApiError on failure', async () => {
      mockErr(401, 'invalid player token');

      const error = await resolveTurn().catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(401);
    });
  });

  describe('surrender', () => {
    const req = { teamId: 0 };
    const fixture: GameEvent[] = [{ type: 'matchEnded', winnerTeamId: 1 }];

    it('should POST surrender request with auth', async () => {
      mockOk(200, fixture);

      const result = await surrender(req);

      expect(result).toEqual(fixture);
      const [, options] = mockFetch.mock.calls[0] as [string, RequestInit];
      expect(options.method).toBe('POST');
      expect(options.headers).toHaveProperty('Authorization');
      expect(options.body).toBe(JSON.stringify(req));
    });

    it('should throw ApiError on invalid team id', async () => {
      mockErr(400, 'invalid game config');

      await expect(surrender({ teamId: 5 })).rejects.toThrow(ApiError);
    });
  });

  describe('getMatchConfig', () => {
    const fixture: GameCfg = makeCfg({
      stagePreset: 'MAP01',
      p1Teams: ['Bomber'],
      p2Teams: ['Bomber'],
      maxTurns: 10,
    });

    it('should GET and return game config', async () => {
      mockOk(200, fixture);

      const result = await getMatchConfig();

      expect(result).toEqual(fixture);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/match-rooms/test-room-123/match/config')
      );
    });

    it('should throw ApiError on failure', async () => {
      mockErr(404, 'match not found');

      await expect(getMatchConfig()).rejects.toThrow(ApiError);
    });
  });

  describe('getAllowedTiles', () => {
    it('should send unitId and turnCmdType as query params', async () => {
      const fixture = [
        { x: 1, y: 2 },
        { x: 2, y: 3 },
      ];
      mockOk(200, fixture);

      const result = await getAllowedTiles({ unitId: 1, turnCmdType: 'move' });

      expect(result).toEqual(fixture);
      const [url] = mockFetch.mock.calls[0] as [string];
      expect(url).toContain('unitId=1');
      expect(url).toContain('turnCmdType=move');
    });

    it('should throw ApiError on failure', async () => {
      mockErr(400, 'missing required query parameters');

      const error = await getAllowedTiles({ unitId: 1, turnCmdType: 'move' }).catch(
        (e: unknown) => e
      );
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).status).toBe(400);
    });
  });
});
