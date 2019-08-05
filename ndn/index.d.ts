export as namespace ndn;

export type Name = string;

export enum NackReason {
  None = 0,
  Congestion = 50,
  Duplicate = 100,
  NoRoute = 150,
  Unspecified = 255,
}
