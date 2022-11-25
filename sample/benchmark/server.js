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

/** @returns {Environment.Parser<readonly string[]>} */
function parsePorts() {
  const parseArray = parsers.array({ parser: parsers.regex(/^[\da-f]{2}:[\da-f]{2}\.[\da-f](?:\+\d+)?$/i) });
  return (s) => {
    const a = parseArray(s);
    if (a.length !== 2) {
      throw new EnvironmentVariableError("expect exactly two Ethernet adapters");
    }
    return a;
  };
}

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
  F_PORTS: { envVarName: "F_PORTS", parser: parsePorts(), required: true },
  F_NUMA_PRIMARY: { envVarName: "F_NUMA_PRIMARY", parser: parsers.nonNegativeInteger, required: true },
  F_CORES_PRIMARY: { envVarName: "F_CORES_PRIMARY", parser: parseCores(5), required: true },
  F_CORES_SECONDARY: { envVarName: "F_CORES_SECONDARY", parser: parseCores(2), required: true },
  G_GQLSERVER: { envVarName: "G_GQLSERVER", parser: parsers.url, required: true },
  G_PORTS: { envVarName: "G_PORTS", parser: parsePorts(), required: true },
  G_NUMA_PRIMARY: { envVarName: "G_NUMA_PRIMARY", parser: parsers.nonNegativeInteger, required: true },
  G_CORES_PRIMARY: { envVarName: "G_CORES_PRIMARY", parser: parseCores(8), required: true },
  G_CORES_SECONDARY: { envVarName: "G_CORES_SECONDARY", parser: parseCores(1), required: true },
  G_FILESERVER_PATH: { envVarName: "G_FILESERVER_PATH", parser: parsers.string, required: true },
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
  { upstream: env.G_GQLSERVER, prefix: "/G" },
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
  G_GQLSERVER: "/G",
}));

await fastify.listen({
  port: 3333,
  host: "127.0.0.1",
});
