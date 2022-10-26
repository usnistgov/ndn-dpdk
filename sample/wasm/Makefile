all: main.wasm wasm_exec.js

main.wasm: main.go
	tinygo build -o $@ -target=wasm ./main.go

wasm_exec.js:
	install --mode 0644 -t $(@D) /usr/local/lib/tinygo/targets/wasm_exec.js
