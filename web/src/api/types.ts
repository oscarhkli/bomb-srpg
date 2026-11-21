import type { components, paths } from './schema.gen';

type S = components['schemas'];

export type TerrainType = S['TerrainType'];
export type OccupantType = S['OccupantType'];
export type SkillType = S['SkillType'];
export type TurnCmdType = S['TurnCmdType'];
export type GameEvtType = S['GameEvtType'];
export type UnitRole = S['UnitRole'];
export type CPUTurnPhase = S['CPUTurnPhase'];

export type Coordinate = S['Coordinate'];
export type Tile = S['Tile'];
export type SoftBlock = S['SoftBlock'];
export type Archetype = S['Archetype'];
export type StagePreset = S['StagePreset'];
export type Catalog = S['Catalog'];
export type Unit = S['Unit'];
export type Bomb = S['Bomb'];
export type TurnCommand = S['TurnCommand'];
export type TeamSlot = S['TeamSlot'];
export type GameCfg = S['GameCfg'];
export type CreateMatchRoomResponse = S['CreateMatchRoomResponse'];
export type CreateMatchRequest = S['CreateMatchRequest'];
export type CreateMatchResponse = S['CreateMatchResponse'];
export type SurrenderRequest = S['SurrenderRequest'];

export type AllowedTilesRequest =
  paths['/api/match-rooms/{roomID}/match/allowed-tiles']['get']['parameters']['query'];
export type AllowedTilesResponse = Coordinate[];

type AlwaysSentOptionalEventKeys = 'unitId' | 'newHp' | 'countdown';

export type GameEvent = Omit<S['GameEvent'], AlwaysSentOptionalEventKeys> &
  Partial<Pick<S['GameEvent'], AlwaysSentOptionalEventKeys>>;

export type GameState = Omit<S['GameState'], 'turnStartEvents'> & {
  turnStartEvents: GameEvent[];
};

export type CpuStatusResponse = Omit<
  S['CpuStatusResponse'],
  'planGameEvents' | 'resolveTurnGameEvents'
> & { planGameEvents: GameEvent[]; resolveTurnGameEvents: GameEvent[] };

export type ResolveTurnResponse = Omit<
  S['ResolveTurnResponse'],
  'planGameEvents' | 'resolveTurnGameEvents'
> & { planGameEvents: GameEvent[]; resolveTurnGameEvents: GameEvent[] };
