# NDN-DPDK Status Page

This is a web application that displays the dynamic status of a running NDN-DPDK service instance.
It is meant as a demonstration on how to access NDN-DPDK GraphQL to gather service status.

## Features

* Forwarder diagram
* Traffic generator diagram
* Ethernet adapters and faces
* Worker threads

## Instructions

1. Run `corepack pnpm install` to install dependencies.

2. Run `corepack pnpm start` to start the web application.

   * If you are running NDN-DPDK in Docker or running multiple instances, use `--gqlserver` flag to specify the GraphQL server.
   * Use `--help` flag to view all command line options.

3. Visit `http://127.0.0.1:3333` (via SSH tunnel) in your browser.
