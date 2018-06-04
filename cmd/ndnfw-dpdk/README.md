# ndnfw-dpdk

This program is a NDN forwarder.

## Usage

```
sudo ndnfw-dpdk EAL-ARGS -- \
  [-initcfg=INITCFG]
```

**-initcfg** accepts an initialization configuration object in YAML format.
This program recognizes *mempool*, *ndt*, *fib*, and *fwdp* sections.
