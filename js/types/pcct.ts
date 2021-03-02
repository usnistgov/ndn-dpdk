/**
 * PIT-CS Composite Table (PCCT) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/pcct#Config>
 */
export interface PcctConfig {
  pcctCapacity?: number;
  csDirectCapacity?: number;
  csIndirectCapacity?: number;
}
