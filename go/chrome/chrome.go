//go:build js && wasm

// Copyright 2017 Google LLC
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

// Package chrome exposes Go versions of Chrome's extension APIs.
package chrome

import (
	"errors"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/dom"
)

// C provides access to Chrome's extension APIs.
type C struct {
	// chrome is a reference to the top-level 'chrome' Javascript object.
	chrome js.Value
	// runtime is a reference to 'chrome.runtime'.
	runtime js.Value
	// syncStorage is a reference to 'chrome.storage.sync'.
	syncStorage js.Value
	// extensionID is the unique ID allocated to our extension.
	extensionID string
}

// New returns an instance of C that can be used to access Chrome's extension
// APIs. Set chrome to nil to access the default Chrome API implementation;
// it should only be overridden for testing.
func New(chrome js.Value) *C {
	if chrome.IsUndefined() || chrome.IsNull() {
		chrome = js.Global().Get("chrome")
	}

	return &C{
		chrome:      chrome,
		runtime:     chrome.Get("runtime"),
		syncStorage: chrome.Get("storage").Get("sync"),
		extensionID: chrome.Get("runtime").Get("id").String(),
	}
}

// SyncStorage returns a PersistentStore object that can be used to to store
// persistent data that is synchronized with Chrome Sync.
//
// See https://developer.chrome.com/apps/storage#property-sync.
func (c *C) SyncStorage() PersistentStore {
	return &BigStorage{
		maxItemBytes: c.syncStorage.Get("QUOTA_BYTES_PER_ITEM").Int(),
		s: &Storage{
			chrome: c,
			o:      c.syncStorage,
		},
	}
}

// OnMessage installs a callback that will be invoked when the extension
// receives a message.
//
// See https://developer.chrome.com/apps/runtime#event-onMessage.
func (c *C) OnMessage(callback func(header js.Value, sender js.Value, sendResponse func(js.Value)) bool) {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var header, sender, sendResponse js.Value
		dom.ExpandArgs(args, &header, &sender, &sendResponse)
		return callback(header, sender, func(rsp js.Value) {
			sendResponse.Invoke(rsp)
		})
	})
	c.runtime.Get("onMessage").Call("addListener", cb)
}

// SendMessage sends a message within our extension. While the underlying
// Chrome API supports sending a message to another extension, we only
// expose functionality to send within the same extension.
//
// See https://developer.chrome.com/apps/runtime#method-sendMessage.
func (c *C) SendMessage(msg js.Value, callback func(rsp js.Value)) {
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer cb.Release() // One-shot callback; release after invoked.
		callback(dom.SingleArg(args))
		return nil
	})
	c.runtime.Call("sendMessage", c.extensionID, msg, nil, cb)
}

// OnConnectExternal installs a callback that will be invoked when an external
// connection is received.
//
// See https://developer.chrome.com/apps/runtime#event-onConnectExternal.
func (c *C) OnConnectExternal(callback func(port js.Value)) {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback(dom.SingleArg(args))
		return nil
	})
	c.runtime.Get("onConnectExternal").Call("addListener", cb)
}

// Error returns the error (if any) from the last call. Returns nil if there
// was no error.
//
// See https://developer.chrome.com/apps/runtime#property-lastError.
func (c *C) Error() error {
	if err := c.runtime.Get("lastError"); !err.IsNull() && !err.IsUndefined() {
		return errors.New(err.Get("message").String())
	}
	return nil
}
