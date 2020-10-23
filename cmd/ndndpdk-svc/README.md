# ndndpdk-svc

This program executes the NDN-DPDK service.
It is intended to be controlled by a service manager, which should automatically restart the process if it exits.

Initially, the program only provides an HTTP GraphQL server at `http://127.0.0.1:3030/`.
The listening endpoint may be overridden via `--gqlserver` command line flag.
You can connect to this GraphQL server and use introspection to discover its schema.

To activate as a forwarder, invoke the `activate` mutation with `forwarder` argument.
The argument is a JSON object that conforms to `ActivateFwArgs` in TypeScript or `build/share/ndn-dpdk/forwarder.schema.json` JSON schema.

To activate as a traffic generator, invoke the `activate` mutation with `trafficgen` argument.
The argument is a JSON object that conforms to `ActivateGenArgs` in TypeScript or `build/share/ndn-dpdk/trafficgen.schema.json` JSON schema.

## Usage

```bash
sudo ndndpdk-svc
node forwarder.config.js | ndndpdk-ctrl activate-forwarder
node trafficgen.config.js | ndndpdk-ctrl activate-trafficgen
```
