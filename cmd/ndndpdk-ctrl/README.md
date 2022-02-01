# ndndpdk-ctrl

Command ndndpdk-ctrl controls the running NDN-DPDK service via GraphQL.
Run `ndndpdk-ctrl help` to show the available subcommands.

Most subcommands print to stdout in [ndjson](https://github.com/ndjson/ndjson-spec) format.
You may use [jq](https://stedolan.github.io/jq/) or [gron](https://github.com/tomnomnom/gron) for further processing.

## GraphQL Schema and Queries

The default GraphQL endpoint of the NDN-DPDK service is `http://127.0.0.1:3030/`.
You may change it by passing the `--gqlserver` flag to both `ndndpdk-svc` and this command.

GraphQL service schema is [published online](https://ndn-dpdk.ndn.today/schema/ndndpdk-svc.graphql).
You can also discover the schema via introspection.
With the NDN-DPDK service running (does not need to be activated), run:

```bash
npx -y graphqurl http://127.0.0.1:3030/ --introspect > ndndpdk-svc.graphql
```

You can view the GraphQL query prepared by this command via the `--cmdout` flag.
For example:

```bash
ndndpdk-ctrl --cmdout show-version
```

Note that the `--gqlserver` and `--cmdout` flags must be specified between `ndndpdk-ctrl` and the subcommand name.
