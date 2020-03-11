export interface LCoreAllocRoleConfig {
  LCores?: number[];
  PerNuma?: { [k: number]: number };
}

export type LCoreAllocConfig = Record<string, LCoreAllocRoleConfig>;
