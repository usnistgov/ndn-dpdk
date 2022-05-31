import type { NNNanoseconds, Uint } from "./core.js";

/**
 * Packet queue configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface#PktQueueConfig>
 */
export type PktQueueConfig = PktQueueConfig.Plain | PktQueueConfig.Delay | PktQueueConfig.CoDel;

export namespace PktQueueConfig {
  interface Common {
    /**
     * @minimum 64
     */
    capacity?: Uint;

    /**
     * @default 64
     * @minimum 1
     * @maximum 64
     */
    dequeueBurstSize?: Uint;
  }

  export interface Plain extends Common {
    disableCoDel: true;
  }

  export interface Delay extends Common {
    delay: NNNanoseconds;
  }

  export interface CoDel extends Common {
    disableCoDel?: false;

    /**
     * @default 5000000
     */
    target?: NNNanoseconds;

    /**
     * @default 100000000
     */
    interval?: NNNanoseconds;
  }
}
