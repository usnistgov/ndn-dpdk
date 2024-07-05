import type { Counter, NNMilliseconds, Uint } from "./core.js";
import type { EthNetifConfig } from "./dpdk.js";

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
export type FaceLocator = FallbackLocator | EtherLocator | UdpLocator | VxlanLocator | GtpLocator | MemifLocator | SocketFaceLocator;

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
export type EthPortConfig = EthNetifConfig & {
  /**
   * @minimum 256
   * @default 4096
   */
  rxQueueSize?: Uint;

  /**
   * @minimum 256
   * @default 4096
   */
  txQueueSize?: Uint;

  /**
   * @minimum 960
   * @maximum 65000
   */
  mtu?: Uint;

  rxFlowQueues?: number;
};

interface EtherLocatorBase extends FaceConfig {
  port?: string;

  /**
   * @minimum 1
   * @maximum 8
   * @default 1
   */
  nRxQueues?: Uint;

  disableTxMultiSegOffload?: boolean;
  disableTxChecksumOffload?: boolean;

  local: string;
  remote: string;

  /**
   * @minimum 1
   * @maximum 4094
   */
  vlan?: Uint;
}

/**
 * Fallback face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#FallbackLocator>
 */
export interface FallbackLocator extends Omit<EtherLocatorBase, "remote" | "vlan"> {
  scheme: "fallback";
}

/**
 * Ethernet face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#EtherLocator>
 */
export interface EtherLocator extends EtherLocatorBase {
  scheme: "ether";
}

interface IpLocatorBase extends EtherLocatorBase {
  localIP: string;
  remoteIP: string;
}

/**
 * @minimum 1
 * @maximum 65535
 */
 type UdpPort = Uint;

/**
 * Ethernet-based UDP face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#UDPLocator>
 */
export interface UdpLocator extends IpLocatorBase {
  scheme: "udpe";
  localUDP: UdpPort;
  remoteUDP: UdpPort;
}

/**
 * VXLAN face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#VxlanLocator>
 */
export interface VxlanLocator extends IpLocatorBase {
  scheme: "vxlan";

  /**
   * @maximum 16777215
   */
  vxlan: Uint;

  innerLocal: string;
  innerRemote: string;
}

/**
 * GTP-U face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/ethface#GtpLocator>
 */
export interface GtpLocator extends IpLocatorBase {
  scheme: "gtp";

  /**
   * @maximum 4294967295
   */
  ulTEID: Uint;

  /**
   * @maximum 63
   */
  ulQFI: Uint;

  /**
   * @maximum 4294967295
   */
  dlTEID: Uint;

  /**
   * @maximum 63
   */
  dlQFI: Uint;

  innerLocalIP: string;
  innerRemoteIP: string;
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

  socketOwner?: [Uint, Uint];

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
 * Socket face global options.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/socketface#GlobalConfig>
 */
export interface SocketFaceGlobalConfig {
  socket?: Uint;
  rxConns?: {
    /**
     * @minimum 64
     * @maximum 65536
     * @default 4096
     */
    ringCapacity?: Uint;
  };
  rxEpoll?: {
    disabled?: boolean;
  };
  txSyscall?: {
    disabled?: boolean;
  };
}

/**
 * Socket face configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/socketface#Config>
 */
export interface SocketFaceConfig extends FaceConfig {
  /**
   * @minimum 960
   * @maximum 65000
   */
  mtu?: Uint;

  /**
   * @default 100
   */
  redialBackoffInitial?: NNMilliseconds;

  /**
   * @default 60000
   */
  redialBackoffMaximum?: NNMilliseconds;
}

/**
 * Socket face locator.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/iface/socketface#Locator>
 */
export interface SocketFaceLocator extends SocketFaceConfig {
  scheme: "udp" | "tcp" | "unix";
  local?: string;
  remote: string;
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
