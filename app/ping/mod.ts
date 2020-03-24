import * as iface from "../../iface/mod";
import { Config as FetchConfig } from "../fetch/mod";
import { Config as ClientConfig } from "../pingclient/mod";
import { Config as ServerConfig } from "../pingserver/mod";

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
