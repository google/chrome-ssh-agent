importScripts('app.js');

const app = new WASMApp("../go/background/background.wasm");

// Workaround for https://github.com/w3c/ServiceWorker/issues/1499#issuecomment-578730536.
// The cited issue illustrates limitation for Rust, but we have the same in Go.
//
// To workaround it, register event handlers at the top-level in Javascript,
// and then forward them into Go.
console.debug('Installing event handlers');

async function onMessageReceived(message, sender, sendResponse) {
	await app.waitInit()
	return handleOnMessage(message, sender, sendResponse);
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
	onMessageReceived(message, sender, sendResponse);
	return true;  // sendResponse invoked asynchronously.
});

async function onConnectionMessage(port, msg) {
	await app.waitInit()
	return handleConnectionMessage(port, msg);
}

async function onConnectionDisconnect(port) {
	await app.waitInit()
	return handleConnectionDisconnect(port);
}

chrome.runtime.onConnectExternal.addListener((port) => {
	// The OnConnectExternal handler must be synchronous in order to
	// guarantee that installed event handlers are in place before the other
	// side of the connection starts sending messages.  Without this, we can
	// miss events.
	port.onMessage.addListener((msg) => onConnectionMessage(port, msg));
	port.onDisconnect.addListener((p) => onConnectionDisconnect(p));
});
