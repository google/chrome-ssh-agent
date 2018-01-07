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
	"github.com/gopherjs/gopherjs/js"
)

type MessageHub struct {
	handlers []func(*js.Object, *js.Object, func(interface{})) bool
}

func NewMessageHub() *MessageHub {
	return &MessageHub{}
}

func (m *MessageHub) OnMessage(callback func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool) {
	m.handlers = append(m.handlers, callback)
}

func (m *MessageHub) SendMessage(msg interface{}, callback func(rsp *js.Object)) {
	for _, h := range m.handlers {
		h(toJSObject(msg), nil, func(rsp interface{}) {
			callback(toJSObject(rsp))
		})
	}
}

func (m *MessageHub) Error() error {
	return nil
}
