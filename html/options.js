const go = new Go();
WebAssembly.instantiateStreaming(fetch("../go/options/options.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
});
