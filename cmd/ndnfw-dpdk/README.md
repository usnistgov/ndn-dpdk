# ndnfw-dpdk

This program is an NDN forwarder.

## Usage

```sh
sudo ndnfw-dpdk EAL-ARGS -- [-initcfg=INITCFG]
```

**-initcfg** accepts an initialization configuration object in YAML format.
This program recognizes the `Mempool`, `Ndt`, `Fib`, and `Fwdp` sections.
See [here](../../docs/init-config.sample.yaml) for an example.
