import type { Counter, Uint } from "../core";
import type { DataGen, Name } from "../ndni";
import type { PktQueueConfig } from "../pktqueue";

/**
 * Traffic generator producer config.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tgproducer#Config>
 */
export interface TgpConfig {
  nThreads?: Uint;
  rxQueue?: PktQueueConfig.Plain | PktQueueConfig.Delay;
  patterns: TgpPattern[];
}

/**
 * Traffic generator producer pattern definition.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tgproducer#Pattern>
 */
export interface TgpPattern {
  prefix: Name;
  replies: TgpReply[];
}

export type TgpReply = TgpReply.Data | TgpReply.Nack | TgpReply.Timeout;

export namespace TgpReply {
  interface Common {
    /**
     * @default 1
     * @minimum 1
     */
    weight?: Uint;
  }

  export interface Data extends Common, DataGen {
  }

  export interface Nack extends Common {
    /**
     * @minimum 1
     * @maximum 255
     */
    nack: Uint;
  }

  export interface Timeout extends Common {
    timeout: true;
  }
}

export interface TgpCounters {
  perPattern: TgpCounters.PatternCounters[];
  nInterests: Counter;
  nNoMatch: Counter;
  nAllocError: Counter;
}
export namespace TgpCounters {
  export interface PatternCounters {
    nInterests: Counter;
    perReply: Counter[];
  }
}
