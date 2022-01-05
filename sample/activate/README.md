# NDN-DPDK Activation Sample

This is a Node.js application that generates activation parameters for NDN-DPDK service.

* [fw-args.ts](fw-args.ts): [forwarder](../docs/forwarder.md) activation parameters.
* [gen-args.ts](gen-args.ts): [forwarder](../docs/forwarder.md) activation parameters.
* [gen-config.ts](gen-config.ts): [forwarder](../docs/forwarder.md) activation parameters.
* [fileserver-args.ts](fileserver-args.ts): [forwarder](../docs/forwarder.md) activation parameters.

## Usage

1. Make a copy of this directory to somewhere outside the NDN-DPDK repository.
2. Run `npm install` to install dependencies.
3. Open the directory in Visual Studio Code or some other editor that recognizes TypeScript definitions.
   If the NDN-DPDK installation is on a remote machine, you may use the Remote-SSH plugin.
4. Open a `.ts` file in the editor, and make changes.
   The editor can provide hints on available options.
5. Run `npm run -s typecheck` to verify your arguments conform to the TypeScript definitions.
6. Run `npm start -s [filename] | jq` to see the JSON document.
7. Run `npm start -s [filename] | ndndpdk-ctrl [subcommand]` to send the activation command to NDN-DPDK.

## Available Samples

[Forwarder](../docs/forwarder.md): activate with `fw-args.ts`

```bash
npm start -s fw-args.ts | ndndpdk-ctrl activate-forwarder
```

[Traffic generator](../docs/trafficgen.md): activate with `gen-args.ts`, use traffic pattern in `gen-config.ts`

```bash
npm start -s gen-args.ts | ndndpdk-ctrl activate-trafficgen
npm start -s gen-config.ts | ndndpdk-ctrl start-trafficgen
```

[File server](../docs/fileserver.md): activate with `fileserver-args.ts`

```bash
npm start -s fileserver-args.ts | ndndpdk-ctrl activate-fileserver
```
