import * as path from "node:path";
import { fileURLToPath } from "node:url";

import FastifyExpress from "@fastify/express";
import FastifyProxy from "@fastify/http-proxy";
import FastifyStatic from "@fastify/static";
import Fastify from "fastify";
import webpack from "webpack";
import devMiddleware from "webpack-dev-middleware";

import { env } from "./env";

export async function serve(port = 3333): Promise<void> {
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
    port,
    host: "127.0.0.1",
  });
}
