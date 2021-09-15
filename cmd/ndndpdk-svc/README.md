# ndndpdk-svc

This program provides the NDN-DPDK service.
It is intended to be controlled by a service manager, which should automatically restart the process if it exits.

Initially, the program only provides an HTTP GraphQL server at `http://127.0.0.1:3030/`.
The listening endpoint may be overridden via `--gqlserver` command line flag.
You can connect to this GraphQL server and use introspection to discover its schema.

To activate the service (as a forwarder or another role), invoke the `activate` mutation with an appropriate argument.
