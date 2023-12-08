# NDN-DPDK Performance Tuning

This page provides some hints on how to maximize NDN-DPDK performance.

## CPU Isolation

NDN-DPDK is a CPU intensive program.
Normally, DPDK pins each worker thread to a CPU core.
This prevents the kernel from moving the thread among different CPU cores, and therefore improves CPU cache locality.
However, the kernel may still place other programs onto the same CPU cores, which would reduce CPU time available to NDN-DPDK.

It is recommended to setup CPU isolation to maximize NDN-DPDK performance on a large server with many CPU cores.
This would assign distinct CPU cores to NDN-DPDK and other programs, so that they do not compete with each other.

To configure CPU isolation:

1. Run `lscpu` and look at "NUMA node*X* CPU(s)" line to determine the available CPU cores on your system.

2. Assign CPU cores to programs other than NDN-DPDK, such as:

    ```bash
    SYSTEM_CPUSET=0-5,8-23
    sudo mkdir -p /etc/systemd/system/init.scope.d
    sudo mkdir -p /etc/systemd/system/service.d
    sudo mkdir -p /etc/systemd/system/user.slice.d
    echo -e "[Scope]\nAllowedCPUs=$SYSTEM_CPUSET" | sudo tee /etc/systemd/system/init.scope.d/cpuset.conf
    echo -e "[Service]\nAllowedCPUs=$SYSTEM_CPUSET" | sudo tee /etc/systemd/system/service.d/cpuset.conf
    echo -e "[Slice]\nAllowedCPUs=$SYSTEM_CPUSET" | sudo tee /etc/systemd/system/user.slice.d/cpuset.conf
    ```

   Generally, the lowest numbered CPU core on each NUMA socket should be assigned to programs other than NDN-DPDK.

3. Assign CPU cores to NDN-DPDK systemd service, such as:

    ```bash
    UNIT_ARG=127.0.0.1:3030
    UNIT_DIR=/etc/systemd/system/ndndpdk-svc@$(systemd-escape $UNIT_ARG).service.d
    UNIT_CPUSET=6-17,24-35
    sudo mkdir -p $UNIT_DIR
    echo -e "[Service]\nAllowedCPUs=$UNIT_CPUSET" | sudo tee $UNIT_DIR/cpuset.conf
    ```

   Repeat this step if you plan to run multiple instances of NDN-DPDK systemd service.
   Each instance should have distinct CPU cores.

   Skip this step if you are running NDN-DPDK as Docker containers.

4. Assign CPU cores to Docker, such as:

    ```bash
    UNIT_CPUSET=6-17,24-35
    sudo mkdir -p /etc/systemd/system/docker-.scope.d
    echo -e "[Scope]\nAllowedCPUs=$UNIT_CPUSET" | sudo tee /etc/systemd/system/docker-.scope.d/cpuset.conf
    ```

   These CPU cores may be used collectively by all Docker containers.
   The CPU cores for each container may be further assigned in a later step.

   Skip this step if you are not using Docker.

5. Reboot the server for the settings to take effect.

6. Assign CPU cores to a Docker container, such as:

    ```bash
    docker run \
      --cpuset-cpus "6-11,24-29" \
      [other arguments]
    ```

   The specified CPU cores must be a subset of CPU cores listed in `/etc/systemd/system/docker-.scope.d/cpuset.conf`.

   Repeat this step if you are running multiple NDN-DPDK Docker containers.
   Each container should have distinct CPU cores.

   Skip this step if you are not running NDN-DPDK as Docker containers.

## LCore Allocation

In DPDK, the main thread as well as each worker thread is referred to as an *lcore*.
Normally, each lcore requires a CPU core (logical processor).
If you need to run NDN-DPDK on a machine with fewer CPU cores, it is possible to map more lcores to fewer CPU cores by setting **.eal.lcoresPerNuma** option in the activation parameters.
NDN-DPDK would run at reduced performance because multiple threads are competing for the same CPU core.
In this case, you may also want to use `NDNDPDK_MK_THREADSLEEP=1` option, see [installation guide](INSTALL.md) "compile-time settings".

When activating the forwarder, you can explicitly specify how many lcores are allocated to each role by setting **.lcoreAlloc** option in the activation parameters.
Example:

```jsonc
{
  "eal": {
    "coresPerNuma": { // 3 CPU cores on each NUMA socket
      "0": 3,
      "1": 3
    },
    "lcoresPerNuma": {
      "0": 5, // 5 lcores on NUMA socket 0, numbered 0,1,2,3,4
      "1": 4  // 4 lcores on NUMA socket 1, numbered 5,6,7,8
    },
    "lcoreMain": 8 // let lcore 8 be the DPDK main lcore
  },
  "lcoreAlloc": { // all roles must be specified unless it has zero lcores
    "RX":     { "0": 1, "1": 1 }, // 1 input thread on each NUMA socket
    "TX":     { "0": 1, "1": 1 }, // 1 output thread on each NUMA socket
    "FWD":    { "0": 3, "1": 0 }, // 3 forwarding threads on NUMA socket 0
    "CRYPTO": { "0": 0, "1": 1 }, // 1 crypto helper thread on NUMA socket 1
  }
  // This snippet is for demonstration purpose. Typically, you should reduce the number of lcores
  // in each role before using .eal.lcoresPerNuma option.
}
```

The traffic generator does not follow **.lcoreAlloc** option.
It always tries to allocate lcores on the same NUMA socket as the Ethernet adapter.

## CPU Usage Insights

Packet processing with DPDK uses continuous polling: every thread runs an endless loop, in which packets (or other items) are retrieved from queues and then processed.
CPU cores used by DPDK always show 100% busy independent of how much work those cores are doing.

NDN-DPDK maintains thread load statistic in polling threads, including these counters:

* empty poll counter, incremented when a thread receives zero packets from its input queue.
* valid poll counter, incremented when a thread receives non-zero packets from its input queue.
* total number of dequeued packets.
* average number of dequeued packets per burst.

These counters can be retrieved using GraphQL subscription `threadLoadStat` with the ID of a worker LCore (found via `workers` query).
Generally, if the empty poll counter of a thread is smaller than its valid poll counter, it indicates the thread is overloaded.

## Memory Usage Insights

When the forwarder or traffic generator is running, with faces created and traffic flowing, you can gain insights in memory usage via GraphQL queries.

Some example queries:

```bash
# declare variable for NDN-DPDK GraphQL endpoint
# if using Docker, see "NDN-DPDK Docker Container" page
GQLSERVER=http://127.0.0.1:3030/

# packet buffers usage
npx -y graphqurl $GQLSERVER -q '{pktmbufPoolTemplates{tid pools{numaSocket used}}}' |\
  jq -c '.data.pktmbufPoolTemplates[] | select(.pools|length>0)'
# This query shows how many objects are currently used in each packet buffer pool.
# You can adjust the packet buffer capacity settings to fit traffic volume.

# memzone report
npx -y graphqurl $GQLSERVER -q '{memoryDiag{memzones}}' | jq -r '.data.memoryDiag.memzones'
# This query shows how DPDK is using hugepages, including size of each memory zone and
# their placement in physical segments (i.e. hugepages).
# You can count how many distinct physical segments are being used, which is useful for
# deciding how many hugepages should be allocated in the system.
```

If you need to run NDN-DPDK on a machine with limited amount of memory, you can try:

1. Set small numbers for packet buffer pool capacity (start with 8192) and FIB/PCCT capacity (start with 1024).
2. Use fewer forwarding threads, because each would create a separate PCCT.
3. Activate the forwarder or traffic generator, and read the usage reports.
4. Change configuration and repeat.
