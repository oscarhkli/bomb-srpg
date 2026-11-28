import type { BombID, SoftBlockID, UnitID } from '../api/types';

export const unitID = (n: number) => n as UnitID;
export const bombID = (n: number) => n as BombID;
export const softBlockID = (n: number) => n as SoftBlockID;
