import * as fib from "../../container/fib";
import * as strategycode from "../../container/strategycode";
import { Counter } from "../../core";
import * as iface from "../../iface";
import * as ndn from "../../ndn";

export as namespace fibmgmt;

export interface FibInfo {
  NEntries: Counter;
  NEntriesDup: Counter;
  NVirtuals: Counter;
  NNodes: Counter;
}

export interface NameArg {
  Name: ndn.Name;
}

export interface InsertArg extends NameArg {
  Nexthops: iface.FaceId[];
  StrategyId: strategycode.Id;
}

export interface InsertReply {
  IsNew: boolean;
}

interface LookupReplyNo {
  HasEntry: false;
}

interface LookupReplyYes {
  HasEntry: true;
  Name: ndn.Name;
  Nexthops: iface.FaceId[];
  StrategyId: strategycode.Id;
}

export type LookupReply = LookupReplyNo | LookupReplyYes;

export interface FibMgmt {
  Info: {args: {}, reply: FibInfo};
  List: {args: {}, reply: ndn.Name[]};
  Insert: {args: InsertArg, reply: InsertReply};
  Erase: {args: NameArg, reply: {}};
  Find: {args: NameArg, reply: LookupReply};
  Lpm: {args: NameArg, reply: LookupReply};
  ReadEntryCounters: {args: NameArg, reply: fib.EntryCounters};
}
