export namespace fibmgmt {
  export interface InsertArg {
    Name: string;
    Nexthops: number[];
    StrategyId?: number;
  }

  export interface InsertRes {
    IsNew: boolean;
  }
}
