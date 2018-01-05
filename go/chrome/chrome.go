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
	chrome *js.Object
}

func New(chrome *js.Object) *C {
	return &C{chrome: chrome}
}

func (c *C) Runtime() *js.Object {
	return c.chrome.Get("runtime")
}

func (c *C) Storage() *js.Object {
	return c.chrome.Get("storage")
}

func (c *C) SyncStorage() *Storage {
	return &Storage{
		chrome: c,
		o:      c.Storage().Get("sync"),
	}
}

func (c *C) OnMessage(callback func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool) {
	c.Runtime().Get("onMessage").Call("addListener", callback)
}

func (c *C) SendMessage(extensionId string, msg interface{}, callback func(rsp *js.Object)) {
	c.Runtime().Call("sendMessage", extensionId, msg, nil, callback)
}

func (c *C) OnConnectExternal(callback func(port *js.Object)) {
	c.Runtime().Get("onConnectExternal").Call("addListener", callback)
}

func (c *C) ExtensionId() string {
	return c.Runtime().Get("id").String()
}

func (c *C) Error() error {
	if err := c.Runtime().Get("lastError"); err != nil && err != js.Undefined {
		return errors.New(err.Get("message").String())
	}
	return nil
}
