import type { FaceLocator } from "../iface";
import type { PingClientConfig } from "./client";
import type { FetchConfig } from "./fetch";
import type { PingServerConfig } from "./server";

export type PingConfig = PingTask[];

export interface PingTask {
  Face: FaceLocator;
  Server?: PingServerConfig & { NThreads?: number };
  Client?: PingClientConfig;
  Fetch?: FetchConfig;
}
