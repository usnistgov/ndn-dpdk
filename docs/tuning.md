# NDN-DPDK Performance Tuning

This page provides some hints on how to maximize NDN-DPDK performance.

## CPU Isolation

NDN-DPDK is a CPU intensive program.
When running on a large server with many CPU cores, DPDK pins each worker thread to a CPU core.
This prevents the kernel from moving the thread among different CPU cores, and therefore improves CPU cache locality.
However, the kernel may still place other programs onto the same CPU cores, which would reduce CPU time available to NDN-DPDK.

It is recommended to setup CPU isolation to maximize NDN-DPDK performance on a large server with many CPU cores.
This would assign distinct CPU cores to NDN-DPDK and other programs, so that they do not compete with each other.

To configure CPU isolation for the NDN-DPDK systemd service:

1. Run `lscpu` and look at "NUMA node? CPU(s)" line to determine the available CPU cores on your system.

2. Run `sudoedit /etc/systemd/system.conf.d/cpuset.conf`, and assign CPU cores to programs other than NDN-DPDK, such as:

    ```ini
    [Manager]
    CPUAffinity=0-5 18-23
    ```

   Generally, the lowest numbered CPU core on each NUMA socket should be assigned to programs other than NDN-DPDK.

3. Run `sudo systemctl edit ndndpdk-svc`, assign CPU cores to NDN-DPDK service, such as:

    ```ini
    [Service]
    CPUAffinity=6-17 24-35
    ```

4. Reboot the server for the settings to take effect.

To configure CPU isolation for the NDN-DPDK Docker container:

1. Following the above instructions to assign CPU cores to non-containerized programs.

2. When launching the NDN-DPDK service container, add the `--cpuset-cpus` flag, such as:

    ```bash
    docker run \
      --cpuset-cpus "6-17,24-35" \
      [other arguments]
    ```

3. When launching other containers, add the `--cpuset-cpus` flag but specify distinct CPU cores.

## CPU Usage Insights

Packet processing with DPDK uses continuous polling: every thread runs an endless loop, in which packets are retrieved from queues and then processed.
CPU cores used by DPDK always show 100% busy independent of how much works those cores are doing.

NDN-DPDK maintains thread load statistic in several types of threads, which includes two counters:

* empty poll counter, incremented when a thread receives zero packets from its input queue.
* valid poll counter, incremented when a thread receives non-zero packets from its input queue.

These counters can be retrieved using GraphQL subscription `threadLoadStat`.
The ID input supports these object types:

* Forwarder's input thread.
* Forwarder's forwarding thread.

## Memory Usage Insights

When the forwarder or traffic generator is running, with faces created and traffic flowing, you can gain insights in memory usage via GraphQL queries.

Some example queries:

```bash
# declare variable for NDN-DPDK GraphQL endpoint
# if using Docker, see "NDN-DPDK Docker Container" page
GQLSERVER=http://127.0.0.1:3030/

# packet buffers usage
gq $GQLSERVER -q '{pktmbufPoolTemplates{tid pools{numaSocket used}}}' |\
  jq -c '.data.pktmbufPoolTemplates[] | select(.pools|length>0)'
# This query shows how many objects are currently used in each packet buffer pool.
# You can adjust the packet buffer capacity settings to fit traffic volume.

# memzone report
gq $GQLSERVER -q '{memoryDiag{memzones}}' | jq -r '.data.memoryDiag.memzones'
# This query shows how DPDK is using hugepages, including size of each memory zone and
# their placement in physical segments (i.e. hugepages).
# You can count how many distinct physical segments are being used, which is useful for
# deciding how many hugepages should be allocated in the system.
```

If you need to run NDN-DPDK on a machine with limited amount of memory, you can try:

1. Set small numbers for packet buffer pool capacity (start with 8192) and table sizes (start with 512).
2. Activate the forwarder or traffic generator, and read the usage reports.
3. Change configuration and repeat.
