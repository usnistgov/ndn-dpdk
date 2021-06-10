import type { DataGen, Name } from "../ndni";

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
     * @TJS-type integer
     * @default 1
     * @minimum 1
     */
    weight?: number;
  }

  export interface Data extends Common, DataGen {
  }

  export interface Nack extends Common {
    /**
     * @TJS-type integer
     * @minimum 1
     * @maximum 255
     */
    nack: number;
  }

  export interface Timeout extends Common {
    timeout: true;
  }
}

export interface TgpCounters {
  perPattern: TgpCounters.PatternCounters[];
  nInterests: number;
  nNoMatch: number;
  nAllocError: number;
}
export namespace TgpCounters {
  export interface PatternCounters {
    nInterests: number;
    perReply: number[];
  }
}
