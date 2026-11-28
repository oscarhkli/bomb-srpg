import type { components, paths } from './schema.gen';

type S = components['schemas'];

export type TerrainType = S['TerrainType'];
export type OccupantType = S['OccupantType'];
export type SkillType = S['SkillType'];
export type TurnCmdType = S['TurnCmdType'];
export type UnitRole = S['UnitRole'];
export type CPUTurnPhase = S['CPUTurnPhase'];

export type Coordinate = S['Coordinate'];
export type Tile = S['Tile'];
export type Archetype = S['Archetype'];
export type StagePreset = S['StagePreset'];
export type Catalog = S['Catalog'];
export type TeamSlot = S['TeamSlot'];
export type GameCfg = S['GameCfg'];
export type CreateMatchRoomResponse = S['CreateMatchRoomResponse'];
export type CreateMatchRequest = S['CreateMatchRequest'];
export type CreateMatchResponse = S['CreateMatchResponse'];
export type SurrenderRequest = S['SurrenderRequest'];

export type AllowedTilesResponse = Coordinate[];

export type UnitID = S['Unit']['id'];
export type BombID = S['Bomb']['id'];
export type SoftBlockID = S['SoftBlock']['id'];

export const SYSTEM_OWNER_ID = 0 as UnitID;
export const KING_SLOT = 0;

export type Unit = S['Unit'];
export type Bomb = S['Bomb'];
export type SoftBlock = S['SoftBlock'];
export type TurnCommand = S['TurnCommand'];
export type GameEvent = S['GameEvent'];
export type GameState = S['GameState'];
export type CpuStatusResponse = S['CpuStatusResponse'];
export type ResolveTurnResponse = S['ResolveTurnResponse'];
export type UnitMovedEvent = S['UnitMovedEvent'];
export type BombPlacedEvent = S['BombPlacedEvent'];
export type MatchEndedEvent = S['MatchEndedEvent'];
export type AllowedTilesRequest =
  paths['/api/match-rooms/{roomID}/match/allowed-tiles']['get']['parameters']['query'];
