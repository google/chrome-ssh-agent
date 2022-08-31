// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import './wasm_exec';

const _global = this;

export class WASMApp {
	// Go functions exposed for application lifecycle.
	//
	// Keep in sync with go/jsutil/app.go
	static _INIT_WAIT_FUNC = 'appInitWaitImpl';
	static _TERMINATE_FUNC = 'appTerminateImpl';

	private _running: Promise<boolean>;

	constructor(path: string) {
		console.debug("Loading WASM app");
		const go = new Go();
		this._running = WebAssembly.instantiateStreaming(fetch(path), go.importObject)
			.then((result) => {
				console.debug('Running WASM app');
				go.run(result.instance);
				return true;
			});
	}

	async _resolveFunc(func: string): Promise<() => void> {
		console.debug(`resolveFunc ${func}: waiting for app to run`);
		await this._running;
		console.debug(`resolveFunc ${func}: waiting for function definition`);
		// Wait until defined. We use a timeout to ensure that other
		// events can proceed. This is important to avoid starving other
		// event handlers that may happen during app initialization.
		while (_global === undefined || _global[func] === undefined) {
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

