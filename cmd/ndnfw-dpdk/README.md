# ndnfw-dpdk

This program is an NDN forwarder.

## Usage

```sh
sudo ndnfw-dpdk [-initcfg=INITCFG]
```

**-initcfg** accepts an initialization configuration object in YAML format.
This program recognizes the `eal`, `Mempool`, `LCoreAlloc`, `Ndt`, `Fib`, and `Fwdp` sections.
See [here](../../docs/init-config.sample.yaml) for an example.
