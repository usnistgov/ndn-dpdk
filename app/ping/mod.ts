import * as iface from "../../iface/mod.js";
import { Config as FetchConfig } from "../fetch/mod.js";
import { Config as ClientConfig } from "../pingclient/mod.js";
import { Config as ServerConfig } from "../pingserver/mod.js";

export type AppConfig = TaskConfig[];

export interface TaskConfig {
  Face: iface.Locator;
  Server?: ServerConfig;
  Client?: ClientConfig;

  /**
   * @TJS-type integer
   * @default 0
   */
  Fetch?: number;
  FetchCfg?: FetchConfig;
}
