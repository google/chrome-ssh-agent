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

// Go functions exposed for application lifecycle.
//
// Keep names in sync with go/jsutil/app.go
interface appHandlers {
	appInitWaitImpl: () => Promise<void>;
	appTerminateImpl: () => Promise<void>;
}

// Manage a Go WASM app.
export class WASMApp {
	private _running: Promise<boolean>;

	// Path is the path to compiled WebAssembly program.
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

	// Object to which Go application installs handlers.
	private get handlers(): appHandlers {
		return self as unknown as appHandlers;
	}

	// Wait for app to initialize, and for a particular condition.
	private async waitForAppInit(cond: () => boolean): Promise<void> {
		console.debug("waitForAppInit: waiting for app to run");
		await this._running;
		
		// Wait for condition. We use a timeout to ensure that other
		// events can proceed. This is important to avoid starving other
		// event handlers that may happen during app initialization.
		console.debug("waitForAppInit: waiting for condition");
		while (!cond()) {
			await new Promise(done => setTimeout(done, 5));
		}
		
		console.debug("waitForAppInit: finished");
	}

	// Wait for application initialization to complete.
	public async waitInit() {
		await this.waitForAppInit(() => this.handlers.appInitWaitImpl !== undefined);
		return this.handlers.appInitWaitImpl();
	}

	// Terminate the application.  This initiates the termination, but does
	// not wait for termination to complete.
	public async terminate() {
		await this.waitForAppInit(() => this.handlers.appTerminateImpl !== undefined);
		return this.handlers.appTerminateImpl();
	}
}

