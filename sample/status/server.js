#!/usr/bin/env node

import FastifyExpress from "@fastify/express";
import FastifyStatic from "@fastify/static";
import Fastify from "fastify";
import httpProxy from "http-proxy";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import webpack from "webpack";
import devMiddleware from "webpack-dev-middleware";

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

const proxy = httpProxy.createProxyServer({
  target: "http://127.0.0.1:3030",
  ws: true,
  ignorePath: true,
});
proxy.on("error", (err) => console.warn(err));
fastify.get("/graphql", (request) => {
  proxy.ws(request.raw, request.socket, request.headers);
});

await fastify.listen(3333, "127.0.0.1");
