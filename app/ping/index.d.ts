import { Config as ClientConfig } from "../pingclient";
import { Config as ServerConfig } from "../pingserver";
import * as iface from "../../iface";

export as namespace ping;

export type AppConfig = TaskConfig[];

export interface TaskConfig {
  Face: iface.Locator;
  Client?: ClientConfig;
  Server?: ServerConfig;
}
