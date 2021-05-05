import type { NNNanoseconds } from "../core";
import type { FaceLocator } from "../iface";
import type { PktQueueConfig } from "../pktqueue";
import type { TgcPattern } from "./consumer";
import type { FetcherConfig } from "./fetch";
import type { TgpPattern } from "./producer";

/**
 * Traffic generator configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tg#Config>
 */
export interface TgConfig {
  face: FaceLocator;
  producer?: {
    rxQueue?: PktQueueConfig.Plain|PktQueueConfig.Delay;
    patterns: TgpPattern[];
    /** @TJS-type integer */
    nThreads?: number;
  };
  consumer?: {
    rxQueue?: PktQueueConfig.Plain|PktQueueConfig.Delay;
    patterns: TgcPattern[];
    interval: NNNanoseconds;
  };
  fetcher?: FetcherConfig;
}
