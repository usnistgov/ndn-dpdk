/**
 * Name Dispatch Table (NDT) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/ndt#Config>
 */
export interface NdtConfig {
  prefixLen?: number;
  capacity?: number;
  sampleInterval?: number;
}
