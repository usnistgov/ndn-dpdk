import * as fib from "../../container/fib/mod";
import * as strategycode from "../../container/strategycode/mod";
import { Counter } from "../../core/mod";
import * as iface from "../../iface/mod";
import * as ndni from "../../ndni/mod";

export interface FibInfo {
  NEntries: Counter;
}

export interface NameArg {
  Name: ndni.Name;
}

export interface InsertArg extends NameArg {
  Nexthops: iface.FaceId[];
  StrategyId?: strategycode.Id;
}

export interface InsertReply {
  IsNew: boolean;
}

interface LookupReplyNo {
  HasEntry: false;
}

interface LookupReplyYes {
  HasEntry: true;
  Name: ndni.Name;
  Nexthops: iface.FaceId[];
  StrategyId: strategycode.Id;
}

export type LookupReply = LookupReplyNo | LookupReplyYes;

export interface FibMgmt {
  Info: {args: {}; reply: FibInfo};
  List: {args: {}; reply: ndni.Name[]};
  Insert: {args: InsertArg; reply: InsertReply};
  Erase: {args: NameArg; reply: {}};
  Find: {args: NameArg; reply: LookupReply};
  Lpm: {args: NameArg; reply: LookupReply};
  ReadEntryCounters: {args: NameArg; reply: fib.EntryCounters};
}
