import type { NNMilliseconds, NNNanoseconds } from "../core";
import type { FaceLocator } from "../iface";
import type { PktQueueConfig } from "../pktqueue";
import type { TgcPattern } from "./consumer";
import type { FetchConfig } from "./fetch";
import type { TgpPattern } from "./producer";

/**
 * Traffic generator configuration.
 */
export interface TgConfig {
  tasks: TgTask[];
  counterInterval?: NNMilliseconds;
}

/**
 * Traffic generator task definition.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tg#TaskConfig>
 */
export interface TgTask {
  face: FaceLocator;
  producer?: {
    rxQueue?: PktQueueConfig.Plain|PktQueueConfig.Delay;
    patterns: TgpPattern[];
    nThreads?: number;
  };
  consumer?: {
    rxQueue?: PktQueueConfig.Plain|PktQueueConfig.Delay;
    patterns: TgcPattern[];
    interval: NNNanoseconds;
  };
  fetch?: FetchConfig;
}
