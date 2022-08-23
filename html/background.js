importScripts('wasm_exec.js');

const go = new Go();
console.debug('Loading WASM');
wasmResult = WebAssembly.instantiateStreaming(fetch("../go/background/background.wasm"), go.importObject);
wasmResult.then((result) => {
	console.debug('Running WASM');
	go.run(result.instance);
});


// Workaround for https://github.com/w3c/ServiceWorker/issues/1499#issuecomment-578730536.
// The cited issue illustrates limitation for Rust, but we have the same in Go.
//
// To workaround it, register event handlers at the top-level in Javascript,
// and then forward them into Go.
console.debug('Installing event handlers');

async function resolveFunc(func) {
	console.debug(`resolveFunc ${func}: waiting for WASM`);
	await wasmResult;
	while (!this[func]) { await null; }  // Wait until defined.
	console.debug(`resolveFunc ${func}: available`);
	return this[func];
}

async function onMessageReceived(message, sender, sendResponse) {
	let f = await resolveFunc('handleOnMessage');
	return f(message, sender, sendResponse);
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
	onMessageReceived(message, sender, sendResponse);
	return true;  // sendResponse invoked asynchronously.
});

async function onConnectExternal(port) {
	let f = await resolveFunc('handleOnConnectExternal');
	return f(port);
}

async function onConnectionMessage(port, msg) {
	let f = await resolveFunc('handleConnectionMessage');
	return f(port, msg);
}

async function onConnectionDisconnect(port) {
	let f = await resolveFunc('handleConnectionDisconnect');
	return f(port);
}

chrome.runtime.onConnectExternal.addListener((port) => {
	// The OnConnectExternal handler must be synchronous in order to
	// guarantee that installed event handlers are in place before the other
	// side of the connection starts sending messages.  Without this, we can
	// miss events.
	onConnectExternal(port);
	port.onMessage.addListener((msg) => onConnectionMessage(port, msg));
	port.onDisconnect.addListener((p) => onConnectionDisconnect(p));
});
