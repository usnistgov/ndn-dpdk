#!/usr/bin/env node

import * as path from "node:path";
import { fileURLToPath } from "node:url";

import FastifyExpress from "@fastify/express";
import FastifyProxy from "@fastify/http-proxy";
import FastifyStatic from "@fastify/static";
import Fastify from "fastify";
import urlParseLax from "url-parse-lax";
import webpack from "webpack";
import devMiddleware from "webpack-dev-middleware";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

const args = yargs(hideBin(process.argv))
  .strict()
  .version(false)
  .option("gqlserver", {
    default: "http://127.0.0.1:3030",
    desc: "NDN-DPDK GraphQL server",
    type: "string",
    coerce(input) {
      return new URL("/", input);
    },
  })
  .option("listen", {
    default: "127.0.0.1:3333",
    desc: "Listen host:port",
    type: "string",
    coerce(input) {
      return urlParseLax(input);
    },
  })
  .parseSync();

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
    enforceExtension: false,
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

await fastify.register(FastifyProxy, {
  upstream: args.gqlserver.toString(),
  prefix: "/graphql",
  rewritePrefix: "/",
  websocket: true,
});

await fastify.listen({
  port: args.listen.port,
  host: args.listen.hostname,
});
