# ndndpdk-ctrl

Command ndndpdk-ctrl controls the running NDN-DPDK daemon via GraphQL.
Execute `ndndpdk-ctrl help` to show the available subcommands.

Most subcommands print to stdout in [ndjson](https://github.com/ndjson/ndjson-spec) format.
You may use [jq](https://stedolan.github.io/jq/) or [gron](https://github.com/tomnomnom/gron) for further processing.
