export type WorkerRole = "RX" | "TX" | "CRYPTO" | "DISK" | "FWD" | "CONSUMER" | "PRODUCER";

export interface Worker<Role extends string = WorkerRole> {
  id: string;
  nid: number;
  role: Role;
  numaSocket: number;
}

export type WorkersByRole = Partial<Record<WorkerRole, Worker[]>>;
