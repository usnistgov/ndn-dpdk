# ndndpdk-ctrl

Command ndndpdk-ctrl controls the running NDN-DPDK service via GraphQL.
Execute `ndndpdk-ctrl help` to show the available subcommands.

Most subcommands print to stdout in [ndjson](https://github.com/ndjson/ndjson-spec) format.
You may use [jq](https://stedolan.github.io/jq/) or [gron](https://github.com/tomnomnom/gron) for further processing.

## GraphQL Schema and Queries

The default GraphQL endpoint NDN-DPDK service is `http://127.0.0.1:3030/`.
You may change it by passing `--gqlserver` flag to `ndndpdk-svc` and this command.

You can discover the GraphQL service schema via introspection.
With NDN-DPDK service running (does not need to be activated):

```bash
gq http://127.0.0.1:3030/ --introspect > ndndpdk-svc.graphql
```

You can view the GraphQL query prepared by this command via `--cmdout` flag.
For example:

```bash
ndndpdk-ctrl --cmdout show-version
```

Note that `--gqlserver` and `--cmdout` flags must be written between `ndndpdk-ctrl` and the subcommand name.
