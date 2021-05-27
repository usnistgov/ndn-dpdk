import type { Counter, NNMilliseconds } from "./core";
import type { VDevNetifConfig } from "./dpdk";

/**
 * Numeric face identifier.
 * @TJS-type integer
 * @minimum 1
 * @maximum 65535
 */
export type FaceID = number;

/**
 * Face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface#Locator>
 */
export type FaceLocator = EtherLocator | UdpLocator | VxlanLocator | MemifLocator | SocketFaceLocator;

/**
 * Face configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface#Config>
 */
export interface FaceConfig {
  /**
   * @TJS-type integer
   * @minimum 4
   * @maximum 8192
   * @default 64
   */
  reassemblerCapacity?: number;

  /**
   * @TJS-type integer
   * @minimum 256
   * @default 1024
   */
  outputQueueSize?: number;

  /**
   * @TJS-type integer
   * @minimum 960
   * @maximum 65000
   */
  mtu?: number;
}

/**
 * Ethernet port configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#PortConfig>
 */
export interface EthPortConfig {
  disableRxFlow?: boolean;

  /**
   * @TJS-type integer
   * @minimum 256
   * @default 8192
   */
  rxQueueSize?: number;

  /**
   * @TJS-type integer
   * @minimum 256
   * @default 8192
   */
  txQueueSize?: number;

  /**
   * @TJS-type integer
   * @minimum 960
   * @maximum 65000
   */
  mtu?: number;

  disableSetMTU?: boolean;
}

interface EtherLocatorBase extends FaceConfig {
  port?: string;
  vdevConfig?: VDevNetifConfig;
  portConfig?: EthPortConfig;

  /**
   * @TJS-type integer
   * @minimum 1
   * @maximum 8
   * @default 1
   */
  maxRxQueues?: number;

  disableTxMultiSegOffload?: boolean;
  disableTxChecksumOffload?: boolean;

  local: string;
  remote: string;

  /**
   * @TJS-type integer
   * @minimum 1
   * @maximum 4095
   */
  vlan?: number;
}

/**
 * Ethernet face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#EtherLocator>
 */
export interface EtherLocator extends EtherLocatorBase {
  scheme: "ether";
}

interface UdpLocatorBase extends EtherLocatorBase {
  localIP: string;
  remoteIP: string;

  /**
   * @TJS-type integer
   * @minimum 1
   * @maximum 65535
   */
  localUDP: number;

  /**
   * @TJS-type integer
   * @minimum 1
   * @maximum 65535
   */
  remoteUDP: number;
}

/**
 * Ethernet-based UDP face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#UDPLocator>
 */
export interface UdpLocator extends UdpLocatorBase {
  scheme: "udpe";
}

/**
 * VXLAN face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#VxlanLocator>
 */
export interface VxlanLocator extends UdpLocatorBase {
  scheme: "vxlan";

  /**
   * @TJS-type integer
   * @minimum 0
   * @maximum 16777215
   */
  vxlan: number;

  innerLocal: string;
  innerRemote: string;
}

/**
 * memif face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#MemifLocator>
 */
export interface MemifLocator {
  scheme: "memif";

  socketName: string;

  /**
   * @TJS-type integer
   * @minimum 0
   * @maximum 4294967295
   */
  id: number;

  /**
   * @TJS-type integer
   * @minimum 512
   * @maximum 65535
   * @default 2048
   */
  dataroom?: number;

  /**
   * @TJS-type integer
   * @minimum 2
   * @maximum 16384
   * @default 1024
   */
  ringCapacity?: number;
}

/**
 * Socket face configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/socketface#Config>
 */
export interface SocketFaceConfig extends FaceConfig {
  /**
   * @TJS-type integer
   * @minimum 256
   * @default 4096
   */
  rxGroupCapacity?: number;

  /** @TJS-type integer */
  rxQueueSize?: number;
  /** @TJS-type integer */
  txQueueSize?: number;
  redialBackoffInitial?: NNMilliseconds;
  redialBackoffMaximum?: NNMilliseconds;
}

/**
 * Socket face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/socketface#Locator>
 */
export interface SocketFaceLocator {
  scheme: "udp"|"tcp"|"unix";
  local?: string;
  remote: string;

  config?: SocketFaceConfig;
}

/**
 * Face counters.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface#Counters>
 */
export interface FaceCounters extends FaceRxCounters, FaceTxCounters {
  rxThreads: FaceRxCounters[];
}

export interface FaceRxCounters {
  rxFrames: Counter;
  rxOctets: Counter;
  rxInterests: Counter;
  rxData: Counter;
  rxNacks: Counter;

  rxDecodeErrs: Counter;
  rxReassPackets: Counter;
  rxReassDrops: Counter;
}

export interface FaceTxCounters {
  txFrames: Counter;
  txOctets: Counter;
  txInterests: Counter;
  txData: Counter;
  txNacks: Counter;

  txFragGood: Counter;
  txFragBad: Counter;
  txAllocErrs: Counter;
  txDropped: Counter;
}
