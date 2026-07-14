import type {
  Archetype,
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
  StartTurnResponse,
} from '../types/api';

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

function buildUrl(path: string, query?: Record<string, string>): string {
  if (!query) {
    return path;
  }
  return `${path}?${new URLSearchParams(query).toString()}`;
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const message = await res.text();
    throw new ApiError(res.status, message);
  }
  return res.json() as Promise<T>;
}

export async function getArchetypes(): Promise<Archetype[]> {
  const res = await fetch(buildUrl('/api/archetypes'));
  return handleResponse<Archetype[]>(res);
}

export async function createMatchRoom(): Promise<CreateMatchRoomResponse> {
  const res = await fetch(buildUrl('/api/match-rooms'), {
    method: 'POST',
  });
  return handleResponse<CreateMatchRoomResponse>(res);
}

export async function createMatch(req: CreateMatchRequest): Promise<CreateMatchResponse> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  return handleResponse<CreateMatchResponse>(res);
}

export async function rematch(): Promise<CreateMatchResponse> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/rematch`), {
    method: 'POST',
  });
  return handleResponse<CreateMatchResponse>(res);
}

export async function getMatchState(): Promise<GameState> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/state`));
  return handleResponse<GameState>(res);
}

export async function startTurn(): Promise<StartTurnResponse> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/start-turn`), {
    method: 'POST',
    headers: authHeaders(),
  });
  return handleResponse<StartTurnResponse>(res);
}

export async function submitTurnCommand(cmd: TurnCommand): Promise<GameEvent[]> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/turn-commands`), {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify(cmd),
  });
  return handleResponse<GameEvent[]>(res);
}

export async function resetTurn(): Promise<GameState> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/reset`), {
    method: 'POST',
    headers: authHeaders(),
  });
  return handleResponse<GameState>(res);
}

export async function resolveTurn(): Promise<GameEvent[]> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/resolve`), {
    method: 'POST',
    headers: authHeaders(),
  });
  return handleResponse<GameEvent[]>(res);
}

export async function surrender(req: SurrenderRequest): Promise<GameEvent[]> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/surrender`), {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  return handleResponse<GameEvent[]>(res);
}

export async function getMatchConfig(): Promise<GameCfg> {
  const roomId = requireRoomId();
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/config`));
  return handleResponse<GameCfg>(res);
}

export async function getAllowedTiles(req: AllowedTilesRequest): Promise<AllowedTilesResponse> {
  const roomId = requireRoomId();
  const query = {
    unitId: String(req.unitId),
    turnCmdType: req.turnCmdType,
  };
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/allowed-tiles`, query));
  return handleResponse<AllowedTilesResponse>(res);
}
