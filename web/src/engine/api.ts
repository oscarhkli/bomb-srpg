import type { Archetype, CreateMatchRoomResponse, CreateMatchRequest, CreateMatchResponse, GameState, TurnCommand, GameEvent, SurrenderRequest, AllowedTilesRequest, AllowedTilesResponse, GameCfg } from '../types/api'

let roomId: string | undefined
let token: string | undefined

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

export function initRoom(id: string): void {
  roomId = id
}

export function initToken(tok: string): void {
  token = tok
}

function requireRoom(): { roomId: string; token: string } {
  if (roomId === undefined || token === undefined) {
    throw new Error('Call initRoom() and initToken() before using room-scoped endpoints')
  }
  return { roomId, token }
}

function authHeaders(): HeadersInit {
  return {
    Authorization: `Bearer ${requireRoom().token}`,
  }
}

function buildUrl(path: string, query?: Record<string, string>): string {
  if (!query) {
    return path
  }
  return `${path}?${new URLSearchParams(query).toString()}`
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const message = await res.text()
    throw new ApiError(res.status, message)
  }
  return res.json() as Promise<T>
}

export async function getArchetypes(): Promise<Archetype[]> {
  const res = await fetch(buildUrl('/api/archetypes'))
  return handleResponse<Archetype[]>(res)
}

export async function createMatchRoom(): Promise<CreateMatchRoomResponse> {
  const res = await fetch(buildUrl('/api/match-rooms'), {
    method: 'POST',
  })
  return handleResponse<CreateMatchRoomResponse>(res)
}

export async function createMatch(req: CreateMatchRequest): Promise<CreateMatchResponse> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  return handleResponse<CreateMatchResponse>(res)
}

export async function getMatchState(): Promise<GameState> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/state`))
  return handleResponse<GameState>(res)
}

export async function startTurn(): Promise<GameState> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/start-turn`), {
    method: 'POST',
    headers: authHeaders(),
  })
  return handleResponse<GameState>(res)
}

export async function submitTurnCommand(cmd: TurnCommand): Promise<GameState> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/turn-commands`), {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify(cmd),
  })
  return handleResponse<GameState>(res)
}

export async function resetTurn(): Promise<GameState> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/reset`), {
    method: 'POST',
    headers: authHeaders(),
  })
  return handleResponse<GameState>(res)
}

export async function resolveTurn(): Promise<GameEvent[]> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/resolve`), {
    method: 'POST',
    headers: authHeaders(),
  })
  return handleResponse<GameEvent[]>(res)
}

export async function surrender(req: SurrenderRequest): Promise<GameEvent[]> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/surrender`), {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  return handleResponse<GameEvent[]>(res)
}

export async function getMatchConfig(): Promise<GameCfg> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/config`))
  return handleResponse<GameCfg>(res)
}

export async function getVictoryResult(): Promise<GameEvent[]> {
  const { roomId } = requireRoom()
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/victory`))
  return handleResponse<GameEvent[]>(res)
}

export async function getAllowedTiles(req: AllowedTilesRequest): Promise<AllowedTilesResponse> {
  const { roomId } = requireRoom()
  const query = {
    unitId: String(req.unitId),
    turnCmdType: req.turnCmdType,
  }
  const res = await fetch(buildUrl(`/api/match-rooms/${roomId}/match/allowed-tiles`, query))
  return handleResponse<AllowedTilesResponse>(res)
}
