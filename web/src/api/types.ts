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

type Brand<B extends string> = number & { readonly __brand: B };

export type UnitID = Brand<'UnitID'>;
export type BombID = Brand<'BombID'>;
export type SoftBlockID = Brand<'SoftBlockID'>;

export const SYSTEM_OWNER_ID = 0 as UnitID;
export const KING_SLOT = 0;

export type Unit = Omit<S['Unit'], 'id'> & { id: UnitID };
export type Bomb = Omit<S['Bomb'], 'id' | 'ownerId'> & { id: BombID; ownerId: UnitID };
export type SoftBlock = Omit<S['SoftBlock'], 'id'> & { id: SoftBlockID };

type MoveCommand = Omit<S['MoveCommand'], 'unitId'> & { unitId: UnitID };
type PlaceBombCommand = Omit<S['PlaceBombCommand'], 'unitId'> & { unitId: UnitID };
export type TurnCommand = MoveCommand | PlaceBombCommand;

export type AllowedTilesRequest = Omit<
  paths['/api/match-rooms/{roomID}/match/allowed-tiles']['get']['parameters']['query'],
  'unitId'
> & { unitId: UnitID };

export type UnitMovedEvent = Omit<S['UnitMovedEvent'], 'unitId'> & { unitId: UnitID };
type UnitDamagedEvent = Omit<S['UnitDamagedEvent'], 'unitId'> & { unitId: UnitID };
type UnitDiedEvent = Omit<S['UnitDiedEvent'], 'unitId'> & { unitId: UnitID };
export type BombPlacedEvent = Omit<S['BombPlacedEvent'], 'unitId' | 'bombId'> & {
  unitId: UnitID;
  bombId: BombID;
};
type BombCountdownUpdatedEvent = Omit<S['BombCountdownUpdatedEvent'], 'bombId'> & {
  bombId: BombID;
};
type BombExplodedEvent = Omit<S['BombExplodedEvent'], 'bombId'> & { bombId: BombID };
type SoftBlockDestroyedEvent = Omit<S['SoftBlockDestroyedEvent'], 'softBlockId'> & {
  softBlockId: SoftBlockID;
};
export type MatchEndedEvent = S['MatchEndedEvent'];

export type GameEvent =
  | UnitMovedEvent
  | UnitDamagedEvent
  | UnitDiedEvent
  | BombPlacedEvent
  | BombCountdownUpdatedEvent
  | BombExplodedEvent
  | SoftBlockDestroyedEvent
  | MatchEndedEvent;

export type GameState = Omit<
  S['GameState'],
  'units' | 'bombs' | 'softBlocks' | 'turnCommands' | 'turnStartEvents'
> & {
  units: Unit[];
  bombs: Bomb[];
  softBlocks: SoftBlock[];
  turnCommands: TurnCommand[];
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
