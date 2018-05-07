# ndn-dpdk/docs/mgmtschema

This directory contains a NodeJS program to generate a schema for the [Management](../../mgmt/) API.
A JSON instance constructed as follows shall validate against this schema:

* The instance is a JSON object with three properties.
* The "method" property is a JSON-RPC method name.
* The "params" property is the params in a JSON-RPC call.
* The "result" property is the result in a JSON-RPC reply.
