import * as iface from "../../iface/mod.js";
import { Config as ClientConfig } from "../pingclient/mod.js";
import { Config as ServerConfig } from "../pingserver/mod.js";

export type AppConfig = TaskConfig[];

export interface TaskConfig {
  Face: iface.Locator;
  Client?: ClientConfig;
  Server?: ServerConfig;
}
