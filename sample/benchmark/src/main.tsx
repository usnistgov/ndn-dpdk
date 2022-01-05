import { h, render } from "preact";

import { App } from "./app";

async function main() {
  render(<App/>, document.body);
}

document.addEventListener("DOMContentLoaded", main);
