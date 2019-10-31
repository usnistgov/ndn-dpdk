import { Config as ClientConfig } from "../pingclient";
import { Config as ServerConfig } from "../pingserver";
import { Counter, NNDuration } from "../../core";
import * as iface from "../../iface";
import * as ndn from "../../ndn";

export as namespace ping;

export type AppConfig = TaskConfig[];

export interface TaskConfig {
  Face: iface.Locator;
  Client?: ClientConfig;
  Server?: ServerConfig;
}
