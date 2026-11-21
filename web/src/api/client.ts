import createClient from 'openapi-fetch';
import type { paths } from './schema.gen';
import type {
  CreateMatchRoomResponse,
  CreateMatchRequest,
  CreateMatchResponse,
  GameState,
  TurnCommand,
  GameEvent,
  SurrenderRequest,
  AllowedTilesRequest,
  AllowedTilesResponse,
  GameCfg,
  Catalog,
  CpuStatusResponse,
  ResolveTurnResponse,
} from './types';

const api = createClient<paths>({
  baseUrl: window.location.origin,
  fetch: async req => {
    const res = await globalThis.fetch(req);
    if (res.ok) {
      return res;
    }
    return new Response(JSON.stringify(await res.text()), res);
  },
});

let roomId: string | undefined;
let token: string | undefined;

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export function initRoom(id: string): void {
  roomId = id;
}

export function initToken(tok: string): void {
  token = tok;
}

function requireRoomId(): string {
  if (roomId === undefined) {
    throw new Error('Call initRoom() before using room-scoped endpoints');
  }
  return roomId;
}

function requireToken(): string {
  if (token === undefined) {
    throw new Error('Call initToken() before using authenticated endpoints');
  }
  return token;
}

function authHeaders(): HeadersInit {
  return {
    Authorization: `Bearer ${requireToken()}`,
  };
}

interface Answer<T> {
  data?: T;
  error?: string | number;
  response: Response;
}

function unwrap<T>({ data, error, response }: Answer<T>): T {
  if (error !== undefined || data === undefined) {
    throw new ApiError(response.status, String(error ?? ''));
  }
  return data;
}

function unwrapVoid({ error, response }: Answer<undefined>): void {
  if (error !== undefined) {
    throw new ApiError(response.status, String(error ?? ''));
  }
}

export async function getCatalog(): Promise<Catalog> {
  return unwrap(await api.GET('/api/catalog'));
}

export async function createMatchRoom(): Promise<CreateMatchRoomResponse> {
  return unwrap(await api.POST('/api/match-rooms'));
}

export async function createMatch(req: CreateMatchRequest): Promise<CreateMatchResponse> {
  const roomID = requireRoomId();
  return unwrap(
    await api.POST('/api/match-rooms/{roomID}/match', { params: { path: { roomID } }, body: req })
  );
}

export async function rematch(): Promise<CreateMatchResponse> {
  const roomID = requireRoomId();
  return unwrap(
    await api.POST('/api/match-rooms/{roomID}/rematch', {
      params: { path: { roomID } },
      headers: authHeaders(),
    })
  );
}

export async function deleteMatch(): Promise<void> {
  const roomID = requireRoomId();
  unwrapVoid(
    await api.DELETE('/api/match-rooms/{roomID}/match', {
      params: { path: { roomID } },
      headers: authHeaders(),
    })
  );
}

export async function getMatchState(): Promise<GameState> {
  const roomID = requireRoomId();
  return unwrap(
    await api.GET('/api/match-rooms/{roomID}/match/state', { params: { path: { roomID } } })
  );
}

export async function consumeCpuStatus(): Promise<CpuStatusResponse> {
  const roomID = requireRoomId();
  return unwrap(
    await api.POST('/api/match-rooms/{roomID}/match/cpu-status/consume', {
      params: { path: { roomID } },
      headers: authHeaders(),
    })
  );
}

export async function submitTurnCommand(cmd: TurnCommand): Promise<GameEvent[]> {
  const roomID = requireRoomId();
  return unwrap(
    await api.POST('/api/match-rooms/{roomID}/match/turn-commands', {
      params: { path: { roomID } },
      headers: authHeaders(),
      body: cmd,
    })
  );
}

export async function resetTurn(): Promise<void> {
  const roomID = requireRoomId();
  unwrapVoid(
    await api.POST('/api/match-rooms/{roomID}/match/reset', {
      params: { path: { roomID } },
      headers: authHeaders(),
    })
  );
}

export async function resolveTurn(): Promise<ResolveTurnResponse> {
  const roomID = requireRoomId();
  return unwrap(
    await api.POST('/api/match-rooms/{roomID}/match/resolve', {
      params: { path: { roomID } },
      headers: authHeaders(),
    })
  );
}

export async function surrender(req: SurrenderRequest): Promise<GameEvent[]> {
  const roomID = requireRoomId();
  return unwrap(
    await api.POST('/api/match-rooms/{roomID}/match/surrender', {
      params: { path: { roomID } },
      headers: authHeaders(),
      body: req,
    })
  );
}

export async function getMatchConfig(): Promise<GameCfg> {
  const roomID = requireRoomId();
  return unwrap(
    await api.GET('/api/match-rooms/{roomID}/match/config', { params: { path: { roomID } } })
  );
}

export async function getAllowedTiles(req: AllowedTilesRequest): Promise<AllowedTilesResponse> {
  const roomID = requireRoomId();
  return unwrap(
    await api.GET('/api/match-rooms/{roomID}/match/allowed-tiles', {
      params: { path: { roomID }, query: req },
    })
  );
}
