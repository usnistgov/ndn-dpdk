import type { FaceLocator } from "../iface";
import type { TgcConfig } from "./consumer";
import type { FetcherConfig } from "./fetch";
import type { FileServerConfig } from "./fileserver";
import type { TgpConfig } from "./producer";

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
