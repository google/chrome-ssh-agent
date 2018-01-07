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

	"github.com/gopherjs/gopherjs/js"
)

// C provides access to Chrome's extension APIs.
type C struct {
	// chrome is a reference to the top-level 'chrome' Javascript object.
	chrome *js.Object
	// runtime is a reference to 'chrome.runtime'.
	runtime *js.Object
	// syncStorage is a reference to 'chrome.storage.sync'.
	syncStorage *js.Object
	// extensionID is the unique ID allocated to our extension.
	extensionID string
}

// New returns an instance of C that can be used to access Chrome's extension
// APIs. Set chrome to nil to access the default Chrome API implementation;
// it should only be overridden for testing.
func New(chrome *js.Object) *C {
	if chrome == nil {
		chrome = js.Global.Get("chrome")
	}

	return &C{
		chrome:      chrome,
		runtime:     chrome.Get("runtime"),
		syncStorage: chrome.Get("storage").Get("sync"),
		extensionID: chrome.Get("runtime").Get("id").String(),
	}
}

// SyncStorage returns a Storage object that can be used to to store
// persistent data that is synchronized with Chrome Sync.
//
// See https://developer.chrome.com/apps/storage#property-sync.
func (c *C) SyncStorage() *Storage {
	return &Storage{
		chrome: c,
		o:      c.syncStorage,
	}
}

// OnMessage installs a callback that will be invoked when the extension
// receives a message.
//
// See https://developer.chrome.com/apps/runtime#event-onMessage.
func (c *C) OnMessage(callback func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool) {
	c.runtime.Get("onMessage").Call("addListener", callback)
}

// SendMessage sends a message within our extension. While the underlying
// Chrome API supports sending a message to another extension, we only
// expose functionality to send within the same extension.
//
// See https://developer.chrome.com/apps/runtime#method-sendMessage.
func (c *C) SendMessage(msg interface{}, callback func(rsp *js.Object)) {
	c.runtime.Call("sendMessage", c.extensionID, msg, nil, callback)
}

// OnConnectExternal installs a callback that will be invoked when an external
// connection is received.
//
// See https://developer.chrome.com/apps/runtime#event-onConnectExternal.
func (c *C) OnConnectExternal(callback func(port *js.Object)) {
	c.runtime.Get("onConnectExternal").Call("addListener", callback)
}

// Error returns the error (if any) from the last call. Returns nil if there
// was no error.
//
// See https://developer.chrome.com/apps/runtime#property-lastError.
func (c *C) Error() error {
	if err := c.runtime.Get("lastError"); err != nil && err != js.Undefined {
		return errors.New(err.Get("message").String())
	}
	return nil
}
