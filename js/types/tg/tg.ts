import type { FaceLocator } from "../iface";
import type { TgcConfig } from "./consumer";
import type { FetcherConfig } from "./fetch";
import type { TgpConfig } from "./producer";

/**
 * Traffic generator configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tg#Config>
 */
export type TgConfig = {
  face: FaceLocator;
} & {
  producer?: TgpConfig;
} & ({
  consumer?: TgcConfig;
} | {
  fetcher?: FetcherConfig;
});
