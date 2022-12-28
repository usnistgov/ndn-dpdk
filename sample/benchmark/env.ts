#!/usr/bin/env node

import "dotenv/config";

import { type Parser, EnvironmentVariableError, makeEnv, parsers } from "@strattadb/environment";

import type { ServerEnv } from "./src/benchmark";

const parsePort = parsers.regex(/^[\da-f]{2}:[\da-f]{2}\.[\da-f](?:\+\d+)?$/i);

function parseCores(min: number): Parser<readonly number[]> {
  const parseArray = parsers.array({ parser: parsers.nonNegativeInteger });
  return (s) => {
    const a = parseArray(s);
    if (a.length < min) {
      throw new EnvironmentVariableError(`expect at least ${min} cores`);
    }
    return a;
  };
}

export const env: ServerEnv = makeEnv({
  F_GQLSERVER: { envVarName: "F_GQLSERVER", parser: parsers.url, required: true },
  F_PORT_A: { envVarName: "F_PORT_A", parser: parsePort, required: true },
  F_PORT_B: { envVarName: "F_PORT_B", parser: parsePort, required: true },
  F_NUMA_PRIMARY: { envVarName: "F_NUMA_PRIMARY", parser: parsers.nonNegativeInteger, required: true },
  F_CORES_PRIMARY: { envVarName: "F_CORES_PRIMARY", parser: parseCores(5), required: true },
  F_CORES_SECONDARY: { envVarName: "F_CORES_SECONDARY", parser: parseCores(2), required: true },
  A_GQLSERVER: { envVarName: "A_GQLSERVER", parser: parsers.url, required: true },
  A_PORT_F: { envVarName: "A_PORT_F", parser: parsePort, required: true },
  A_NUMA_PRIMARY: { envVarName: "A_NUMA_PRIMARY", parser: parsers.nonNegativeInteger, required: true },
  A_CORES_PRIMARY: { envVarName: "A_CORES_PRIMARY", parser: parseCores(5), required: true },
  A_CORES_SECONDARY: { envVarName: "A_CORES_SECONDARY", parser: parseCores(1), required: true },
  A_FILESERVER_PATH: { envVarName: "A_FILESERVER_PATH", parser: parsers.string, required: true },
  B_GQLSERVER: { envVarName: "B_GQLSERVER", parser: parsers.url, required: true },
  B_PORT_F: { envVarName: "B_PORT_F", parser: parsePort, required: true },
  B_NUMA_PRIMARY: { envVarName: "B_NUMA_PRIMARY", parser: parsers.nonNegativeInteger, required: true },
  B_CORES_PRIMARY: { envVarName: "B_CORES_PRIMARY", parser: parseCores(5), required: true },
  B_CORES_SECONDARY: { envVarName: "B_CORES_SECONDARY", parser: parseCores(1), required: true },
  B_FILESERVER_PATH: { envVarName: "B_FILESERVER_PATH", parser: parsers.string, required: true },
});
