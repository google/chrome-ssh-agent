//go:build js && wasm

// Copyright 2018 Google LLC
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

package fakes

import (
	"syscall/js"
)

// MessageHub is a fake implementation of Chrome's messaging APIs.
type MessageHub struct {
	handlers []func(js.Value, js.Value, func(js.Value)) bool
}

// NewMessageHub returns a fake implementation of Chrome's messaging APIs.
func NewMessageHub() *MessageHub {
	return &MessageHub{}
}

// OnMessage is a fake implementation of chrome.C.OnMessage.
func (m *MessageHub) OnMessage(callback func(header js.Value, sender js.Value, sendResponse func(js.Value)) bool) {
	m.handlers = append(m.handlers, callback)
}

// SendMessage is a fake implementation of chrome.C.SendMessage.
func (m *MessageHub) SendMessage(msg js.Value, callback func(rsp js.Value)) {
	for _, h := range m.handlers {
		h(msg, js.Null(), func(rsp js.Value) {
			callback(rsp)
		})
	}
}

// Error is a fake implementation of chrome.C.Error. This fake implementation
// does not simulate errors, so Error() always returns nil.
func (m *MessageHub) Error() error {
	return nil
}
