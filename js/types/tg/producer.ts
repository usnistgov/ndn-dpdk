import type { Name, NNMilliseconds } from "../core";
import type { PktQueueConfig } from "../pktqueue";

export interface TgProducerConfig {
  rxQueue?: PktQueueConfig.Plain|PktQueueConfig.Delay;
  patterns: TgProducerPattern[];
  nack?: boolean;
}

export interface TgProducerPattern {
  prefix: Name;
  replies: TgProducerReply[];
}

export type TgProducerReply = TgProducerReply.Data | TgProducerReply.Nack | TgProducerReply.Timeout;

export namespace TgProducerReply {
  interface Common {
    /**
     * @TJS-type integer
     * @default 1
     * @minimum 1
     */
    weight?: number;
  }

  export interface Data extends Common {
    suffix?: Name;

    /**
     * @default 0
     */
    freshnessPeriod?: NNMilliseconds;

    /**
     * @TJS-type integer
     * @default 0
     * @minimum 0
     */
    payloadLen?: number;
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

export interface TgProducerCounters {
  perPattern: TgProducerCounters.PatternCounters[];
  nInterests: number;
  nNoMatch: number;
  nAllocError: number;
}
export namespace TgProducerCounters {
  export interface PatternCounters {
    nInterests: number;
    perReply: number[];
  }
}
