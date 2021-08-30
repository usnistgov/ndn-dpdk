import type { Counter, NNMilliseconds, Uint } from "./core";
import type { VDevNetifConfig } from "./dpdk";

/**
 * Numeric face identifier.
 * @minimum 1
 * @maximum 65535
 */
export type FaceID = Uint;

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
   * @minimum 4
   * @maximum 8192
   * @default 64
   */
  reassemblerCapacity?: Uint;

  /**
   * @minimum 256
   * @default 1024
   */
  outputQueueSize?: Uint;

  /**
   * @minimum 960
   * @maximum 65000
   */
  mtu?: Uint;
}

/**
 * Ethernet port configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#PortConfig>
 */
export interface EthPortConfig {
  disableRxFlow?: boolean;

  /**
   * @minimum 256
   * @default 8192
   */
  rxQueueSize?: Uint;

  /**
   * @minimum 256
   * @default 8192
   */
  txQueueSize?: Uint;

  /**
   * @minimum 960
   * @maximum 65000
   */
  mtu?: Uint;

  disableSetMTU?: boolean;
}

interface EtherLocatorBase extends FaceConfig {
  port?: string;
  vdevConfig?: VDevNetifConfig;
  portConfig?: EthPortConfig;

  /**
   * @minimum 1
   * @maximum 8
   * @default 1
   */
  maxRxQueues?: Uint;

  disableTxMultiSegOffload?: boolean;
  disableTxChecksumOffload?: boolean;

  local: string;
  remote: string;

  /**
   * @minimum 1
   * @maximum 4095
   */
  vlan?: Uint;
}

/**
 * Ethernet face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#EtherLocator>
 */
export interface EtherLocator extends EtherLocatorBase {
  scheme: "ether";
}

/**
 * @minimum 1
 * @maximum 65535
 */
type UdpPort = Uint;

interface UdpLocatorBase extends EtherLocatorBase {
  localIP: string;
  remoteIP: string;
  localUDP: UdpPort;
  remoteUDP: UdpPort;
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
   * @minimum 0
   * @maximum 16777215
   */
  vxlan: Uint;

  innerLocal: string;
  innerRemote: string;
}

export type MemifRole = "server" | "client";

/**
 * memif face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#MemifLocator>
 */
export interface MemifLocator {
  scheme: "memif";

  role?: MemifRole;

  socketName: string;

  /**
   * @minimum 0
   * @maximum 4294967295
   */
  id: Uint;

  /**
   * @minimum 512
   * @maximum 65535
   * @default 2048
   */
  dataroom?: Uint;

  /**
   * @minimum 2
   * @maximum 16384
   * @default 1024
   */
  ringCapacity?: Uint;
}

/**
 * Socket face configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/socketface#Config>
 */
export interface SocketFaceConfig extends FaceConfig {
  /**
   * @minimum 256
   * @default 4096
   */
  rxGroupCapacity?: Uint;

  rxQueueSize?: Uint;
  txQueueSize?: Uint;
  redialBackoffInitial?: NNMilliseconds;
  redialBackoffMaximum?: NNMilliseconds;
}

/**
 * Socket face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/socketface#Locator>
 */
export interface SocketFaceLocator {
  scheme: "udp" | "tcp" | "unix";
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
