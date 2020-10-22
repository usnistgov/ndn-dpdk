import type { NNMilliseconds } from "../core";
import type { FaceLocator } from "../iface";
import type { TgConsumerConfig } from "./consumer";
import type { FetchConfig } from "./fetch";
import type { TgProducerConfig } from "./producer";

export interface TgConfig {
  tasks: TgTask[];
  counterInterval?: NNMilliseconds;
}

export interface TgTask {
  face: FaceLocator;
  producer?: TgProducerConfig & { nThreads?: number };
  consumer?: TgConsumerConfig;
  fetch?: FetchConfig;
}
