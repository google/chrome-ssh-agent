// Gather imports when running within a Web Worker.  In other contexts (e.g.,
// loading in an HTML document), caller is responsible for importing.
if ('function' === typeof importScripts) {
	importScripts('wasm_exec.js');
}

const _global = this;

WASMApp = class {
	// Go functions exposed for application lifecycle.
	//
	// Keep in sync with go/jsutil/app.go
	static _INIT_WAIT_FUNC = 'appInitWaitImpl';
	static _TERMINATE_FUNC = 'appTerminateImpl';

	constructor(path) {
		console.debug("Loading WASM");
		const go = new Go();
		this._wasm = WebAssembly.instantiateStreaming(
			fetch(path),
			go.importObject);
		this._wasm.then((result) => {
			console.debug('Running WASM');
			go.run(result.instance);
		});
	}

	async _resolveFunc(func) {
		console.debug(`resolveFunc ${func}: waiting for WASM`);
		await this._wasm;
		console.debug(`resolveFunc ${func}: waiting for function definition`);
		// Wait until defined. We use a timeout to ensure that other
		// events can proceed. This is important to avoid starving other
		// event handlers that may happen during app initialization.
		while (!_global[func]) {
			await new Promise(done => setTimeout(done, 5));
		}
		console.debug(`resolveFunc ${func}: function definition available`);
		return _global[func];
	}

	async waitInit() {
		const f = await this._resolveFunc(WASMApp._INIT_WAIT_FUNC);
		return f();
	}

	async terminate() {
		const f = await this._resolveFunc(WASMApp._TERMINATE_FUNC);
		return f();
	}
}

