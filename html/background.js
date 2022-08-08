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

async function onMessageReceived(message, sender, sendResponse) {
	console.debug('onMessage: waiting for WASM');
	await wasmResult;
	console.debug('onMessage: waiting for event hook');
	while (!handleOnMessage) { await null; }  // Wait until defined.
	console.debug('onMessage: forwarding event');
	handleOnMessage(message, sender, sendResponse);
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
	onMessageReceived(message, sender, sendResponse);
	return true;  // sendResponse invoked asynchronously.
});

chrome.runtime.onConnectExternal.addListener(async (port) => {
	console.debug('onConnectExternal: waiting for WASM');
	await wasmResult;
	console.debug('onConnectExternal: waiting for event hook');
	while (!handleOnConnectExternal) { await null; }  // Wait until defined.
	console.debug('onConnectExternal: forwarding event');
	handleOnConnectExternal(port);
});

