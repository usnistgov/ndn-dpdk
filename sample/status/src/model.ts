import type { FaceLocator } from "@usnistgov/ndn-dpdk";

export type WorkerRole = "RX" | "TX" | "CRYPTO" | "DISK" | "FWD" | "CONSUMER" | "PRODUCER";

export interface Worker<Role extends string = WorkerRole> {
  id: string;
  nid: number;
  role: Role;
  numaSocket: number;
}
export namespace Worker {
  export const subselection = "id nid role numaSocket";
}

export type WorkersByRole = Partial<Record<WorkerRole, Worker[]>>;

export interface Face {
  id: string;
  nid: number;
  locator: FaceLocator;
  isDown: boolean;
}
export namespace Face {
  export const subselection = "id nid locator isDown";
}

export function describeFaceLocator(loc: FaceLocator): string {
  switch (loc.scheme) {
    case "ether":
      return `Ethernet ${loc.remote}${loc.vlan && ` VLAN ${loc.vlan}`}`;
    case "udpe":
      return `UDP [${loc.remoteIP}]:${loc.remoteUDP}`;
    case "vxlan":
      return `VXLAN ${loc.remoteIP} ${loc.vxlan}`;
    case "unix":
    case "udp":
    case "tcp":
      return `${loc.scheme.toUpperCase()} socket ${loc.remote}`;
    case "memif":
      return `memif ${loc.socketName} ${loc.id}`;
    default:
      return JSON.stringify(loc);
  }
}

export function formatName(name: string): string {
  return name.replaceAll("/8=", "/");
}
