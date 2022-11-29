#!/usr/bin/env node

import "dotenv/config"; // eslint-disable-line import/no-unassigned-import

import * as path from "node:path";
import { fileURLToPath } from "node:url";

import FastifyExpress from "@fastify/express";
import FastifyProxy from "@fastify/http-proxy";
import FastifyStatic from "@fastify/static";
import Environment from "@strattadb/environment";
import Fastify from "fastify";
import webpack from "webpack";
import devMiddleware from "webpack-dev-middleware";

const { makeEnv, parsers, EnvironmentVariableError } = Environment;

const parsePort = parsers.regex(/^[\da-f]{2}:[\da-f]{2}\.[\da-f](?:\+\d+)?$/i);

/**
 * @param {number} min
 * @returns {Environment.Parser<readonly number[]>}
 */
function parseCores(min) {
  const parseArray = parsers.array({ parser: parsers.nonNegativeInteger });
  return (s) => {
    const a = parseArray(s);
    if (a.length < min) {
      throw new EnvironmentVariableError(`expect at least ${min} cores`);
    }
    return a;
  };
}

const env = makeEnv({
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

const publicDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "public");
const compiler = webpack({
  mode: "development",
  devtool: "cheap-module-source-map",
  entry: "./src/main.tsx",
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        exclude: /node_modules/,
        loader: "ts-loader",
      },
    ],
  },
  resolve: {
    extensions: [".tsx", ".ts", ".js"],
  },
  output: {
    filename: "bundle.js",
    path: publicDir,
  },
});

const fastify = Fastify();

await fastify.register(FastifyExpress);
fastify.use(devMiddleware(compiler));

await fastify.register(FastifyStatic, { root: publicDir });

for (const u of [
  { upstream: env.F_GQLSERVER, prefix: "/F" },
  { upstream: env.A_GQLSERVER, prefix: "/A" },
  { upstream: env.B_GQLSERVER, prefix: "/B" },
]) {
  await fastify.register(FastifyProxy, {
    ...u,
    rewritePrefix: "/",
    websocket: true,
  });
}

fastify.get("/env.json", () => ({
  ...env,
  F_GQLSERVER: "/F",
  A_GQLSERVER: "/A",
  B_GQLSERVER: env.A_GQLSERVER === env.B_GQLSERVER ? "/A" : "/B",
}));

await fastify.listen({
  port: 3333,
  host: "127.0.0.1",
});
