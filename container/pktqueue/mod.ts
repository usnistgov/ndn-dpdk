import { Nanoseconds } from "../../core/nnduration/mod.js";

interface ConfigCommon {
  /**
   * @TJS-type integer
   * @minimum 64
   */
  Capacity?: number;

  /**
   * @TJS-type integer
   * @default 64
   * @minimum 1
   * @maximum 64
   */
  DequeueBurstSize?: number;
}

interface ConfigPlain extends ConfigCommon {
  DisableCoDel: true;
}

interface ConfigDelay extends ConfigCommon {
  Delay: Nanoseconds;
}

interface ConfigCoDel extends ConfigCommon {
  DisableCoDel?: false;

  /**
   * @default 5000000
   */
  Target?: Nanoseconds;

  /**
   * @default 100000000
   */
  Interval?: Nanoseconds;
}

export type Config = ConfigPlain | ConfigDelay | ConfigCoDel;
