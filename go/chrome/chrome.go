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

package chrome

import (
	"errors"

	"github.com/gopherjs/gopherjs/js"
)

var (
	Chrome = js.Global.Get("chrome")
)

type C struct {
	chrome      *js.Object
	runtime     *js.Object
	syncStorage *js.Object
}

func New(chrome *js.Object) *C {
	return &C{
		chrome:      chrome,
		runtime:     chrome.Get("runtime"),
		syncStorage: chrome.Get("storage").Get("sync"),
	}
}

func (c *C) SyncStorage() *Storage {
	return &Storage{
		chrome: c,
		o:      c.syncStorage,
	}
}

func (c *C) OnMessage(callback func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool) {
	c.runtime.Get("onMessage").Call("addListener", callback)
}

func (c *C) SendMessage(extensionId string, msg interface{}, callback func(rsp *js.Object)) {
	c.runtime.Call("sendMessage", extensionId, msg, nil, callback)
}

func (c *C) OnConnectExternal(callback func(port *js.Object)) {
	c.runtime.Get("onConnectExternal").Call("addListener", callback)
}

func (c *C) ExtensionId() string {
	return c.runtime.Get("id").String()
}

func (c *C) Error() error {
	if err := c.runtime.Get("lastError"); err != nil && err != js.Undefined {
		return errors.New(err.Get("message").String())
	}
	return nil
}
