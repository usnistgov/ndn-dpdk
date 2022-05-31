import type { FaceLocator } from "../iface.js";
import type { TgcConfig } from "./consumer.js";
import type { FetcherConfig } from "./fetch.js";
import type { FileServerConfig } from "./fileserver.js";
import type { TgpConfig } from "./producer.js";

/**
 * Traffic generator configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tg#Config>
 */
export type TgConfig = {
  face: FaceLocator;
} & ({
  producer?: TgpConfig;
} | {
  fileServer?: FileServerConfig;
}) & ({
  consumer?: TgcConfig;
} | {
  fetcher?: FetcherConfig;
});
