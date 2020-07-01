import type { Counter, Index, Name } from "../core";
import type { FibEntryCounters } from "../fib";
import type { FaceID } from "../iface";
import type { NameArg } from "./common";

export interface FibMgmt {
  Info: {args: {}; reply: FibInfo};
  List: {args: {}; reply: Name[]};
  Insert: {args: FibInsertArg; reply: FibInsertReply};
  Erase: {args: NameArg; reply: {}};
  Find: {args: NameArg; reply: FibLookupReply};
  Lpm: {args: NameArg; reply: FibLookupReply};
  ReadEntryCounters: {args: NameArg; reply: FibEntryCounters};
}

export interface FibInfo {
  NEntries: Counter;
}

export interface FibInsertArg extends NameArg {
  Nexthops: FaceID[];
  StrategyId?: Index;
}

export interface FibInsertReply {
  IsNew: boolean;
}

export type FibLookupReply = FibLookupReply.No | FibLookupReply.Yes;

export namespace FibLookupReply {
  export interface No {
    HasEntry: false;
  }

  export interface Yes {
    HasEntry: true;
    Name: Name;
    Nexthops: FaceID[];
    StrategyId: Index;
  }
}
