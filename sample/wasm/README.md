# NDNgo WebAssembly demo

This is a proof-of-concept of NDNgo library compiled to WebAssembly.

## Instructions

1. Install [TinyGo](https://tinygo.org/) compiler.

2. Run `make` to build the WebAssembly module.

3. Run `python3 -m http.server` to start HTTP server.
   Python 3.8 or later is required for serving .wasm with correct MIME type.

4. Open the webpage in a browser.
