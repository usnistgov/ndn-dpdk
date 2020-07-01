import type { Name, NNMilliseconds } from "../core";
import type { PktQueueConfig } from "../pktqueue";

export interface PingServerConfig {
  RxQueue?: PktQueueConfig.Plain|PktQueueConfig.Delay;
  Patterns: PingServerPattern[];
  Nack: boolean;
}

export interface PingServerPattern {
  Prefix: Name;
  Replies: PingServerReply[];
}

export type PingServerReply = PingServerReply.Data | PingServerReply.Nack | PingServerReply.Timeout;

export namespace PingServerReply {
  interface Common {
    /**
       * @TJS-type integer
       * @default 1
       * @minimum 1
       */
    Weight?: number;
  }

  export interface Data extends Common {
    Suffix: Name;

    /**
       * @default 0
       */
    FreshnessPeriod?: NNMilliseconds;

    /**
       * @TJS-type integer
       * @default 0
       * @minimum 0
       */
    PayloadLen?: number;
  }

  export interface Nack extends Common {
    /**
       * @TJS-type integer
       * @minimum 1
       * @maximum 255
       */
    Nack: number;
  }

  export interface Timeout extends Common {
    Timeout: true;
  }
}
