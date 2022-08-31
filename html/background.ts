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

import {WASMApp} from './app';

const app = new WASMApp("../go/background/background.wasm");

// Declare types for functions exported by background.wasm.
declare function handleOnMessage(message: any, sender: chrome.runtime.MessageSender, sendResponse: (message: any) => void): Promise<void>;
declare function handleConnectionMessage(port: chrome.runtime.Port, message: any): Promise<void>;
declare function handleConnectionDisconnect(port: chrome.runtime.Port): Promise<void>;

// Workaround for https://github.com/w3c/ServiceWorker/issues/1499#issuecomment-578730536.
// The cited issue illustrates limitation for Rust, but we have the same in Go.
//
// To workaround it, register event handlers at the top-level in Javascript,
// and then forward them into Go.
console.debug('Installing event handlers');

async function onMessageReceived(message: any, sender: chrome.runtime.MessageSender, sendResponse: (message: any) => void) {
	await app.waitInit()
	return handleOnMessage(message, sender, sendResponse);
}

chrome.runtime.onMessage.addListener((message: any, sender: chrome.runtime.MessageSender, sendResponse: (message: any) => void) => {
	onMessageReceived(message, sender, sendResponse);
	return true;  // sendResponse invoked asynchronously.
});

async function onConnectionMessage(port: chrome.runtime.Port, msg: any) {
	await app.waitInit()
	return handleConnectionMessage(port, msg);
}

async function onConnectionDisconnect(port: chrome.runtime.Port) {
	await app.waitInit()
	return handleConnectionDisconnect(port);
}

chrome.runtime.onConnectExternal.addListener((port: chrome.runtime.Port) => {
	// The OnConnectExternal handler must be synchronous in order to
	// guarantee that installed event handlers are in place before the other
	// side of the connection starts sending messages.  Without this, we can
	// miss events.
	port.onMessage.addListener((msg: any) => onConnectionMessage(port, msg));
	port.onDisconnect.addListener((port: chrome.runtime.Port) => onConnectionDisconnect(port));
});
