import type { NNNanoseconds, Ratio, Uint } from "../core";
import type { Name } from "../ndni";
import type { PktQueueConfig } from "../pktqueue";

/**
 * File server config.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/fileserver#Config>
 */
export interface FileServerConfig {
  nThreads?: Uint;
  rxQueue?: PktQueueConfig.Plain | PktQueueConfig.Delay;
  mounts: FileServerMount[];
  segmentLen?: Uint;
  uringCapacity?: Uint;
  uringCongestionThres?: Ratio;
  uringWaitThres?: Ratio;
  openFds?: Uint;
  keepFds?: Uint;
  statValidity?: NNNanoseconds;
}

/**
 * File server mount definition.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/fileserver#Config>
 */
export interface FileServerMount {
  prefix: Name;
  path: string;
}
